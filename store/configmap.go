package store

import (
	"context"
	"fmt"
	"os"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"

	v1 "github.com/sttts/sttts-bot/store/v1"
)

type Store interface {
	// pass the old state and let tx update it to the new state, which then is
	// is written to the store. Tx might be called multiple times until success.
	UpdateState(tx func(old *v1.State) (*v1.State, error)) error
	// gives the function read access to the state, locking out any write
	// requests in parallel.
	ReadState(func(*v1.State))
}

type configMapStore struct {
	client   *kubernetes.Clientset
	ns, name string

	lock  sync.RWMutex
	state *v1.State
	cm    *corev1.ConfigMap
}

func NewConfigMapStore(ns, name string) (*configMapStore, error) {
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	s := &configMapStore{
		client: client,
		ns:     ns,
		name:   name,
	}

	cm, state, err := s.read()
	if errors.IsNotFound(err) {
		s.state = &v1.State{}
	} else if err != nil {
		return nil, err
	}

	s.state = state
	s.cm = cm

	return s, nil
}

func (s *configMapStore) read() (*corev1.ConfigMap, *v1.State, error) {
	cm, err := s.client.CoreV1().ConfigMaps(s.ns).Get(context.TODO(), s.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	bs, ok := cm.Data["state.yaml"]
	if !ok {
		return nil, nil, fmt.Errorf(`cannot find "state.yaml" in ConfigMap %s/%s: %v`, cm.Namespace, cm.Name, err)
	}

	state := v1.State{}
	if err := yaml.Unmarshal([]byte(bs), &state); err != nil {
		return nil, nil, fmt.Errorf("failed decoding state from ConfigMap %s/%s: %v", cm.Namespace, cm.Name, err)
	}

	return cm, &state, nil
}

func (s *configMapStore) UpdateState(tx func(old *v1.State) (*v1.State, error)) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	state := s.state
	var lastErr error
	for i := 0; i < 3; i++ {
		newState, err := tx(state)
		if err != nil {
			return err
		}

		cm := s.cm
		if cm == nil {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: s.ns,
					Name:      s.name,
				},
			}
		}

		bs, err := yaml.Marshal(newState)
		if err != nil {
			return fmt.Errorf("failed to encode state while updating: %v", err)
		}
		cm.Data["state.yaml"] = string(bs)

		var updated *corev1.ConfigMap
		updated, lastErr = s.client.CoreV1().ConfigMaps(s.ns).Update(context.TODO(), cm, metav1.UpdateOptions{})
		if errors.IsNotFound(err) {
			updated, lastErr = s.client.CoreV1().ConfigMaps(s.ns).Create(context.TODO(), cm, metav1.CreateOptions{})
		}
		if lastErr != nil && !errors.IsConflict(lastErr) {
			return fmt.Errorf("failed to update ConfigMap %s/%s: %v", s.ns, s.name, lastErr)
		} else if lastErr == nil {
			s.cm = updated
			s.state = newState
			return nil
		}

		cm, state, err = s.read()
		if err != nil {
			return err
		}
	}

	return lastErr
}

func (s *configMapStore) ReadState(process func(*v1.State)) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	process(s.state)
}

var store Store

func UpdateState(tx func(old *v1.State) (*v1.State, error)) error {
	return store.UpdateState(tx)
}

func ReadState(process func(*v1.State)) {
	store.ReadState(process)
}
func init() {
	ns := "default"
	if env := os.Getenv("NAMESPACE"); env != "" {
		ns = env
	}

	var err error
	store, err = NewConfigMapStore(ns, "state")
	if err != nil {
		klog.Fatal(err)
	}
}

package bugzilla

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"k8s.io/klog"
)

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	ID     uint64         `json:"id"`
}

type bugzillaError struct {
	Message string
	Code    uint64
}

type clientResponse struct {
	ID     uint64           `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  *bugzillaError   `json:"error"`
}

// bugzillaJSONRPCClient bugzilla JSON RPC client
type bugzillaJSONRPCClient struct {
	bugzillaAddr                    string
	jsonRPCAddr                     string
	httpClient                      *http.Client
	seq                             uint64
	m                               sync.RWMutex
	bugzillaLogin, bugzillaPassword string
	token                           string
}

// newJSONRPCClient creates a helper json rpc client for regular HTTP based endpoints
func newJSONRPCClient(addr string, httpClient *http.Client, bugzillaLogin, bugzillaPassword string) (*bugzillaJSONRPCClient, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	u.Path = "jsonrpc.cgi"

	return &bugzillaJSONRPCClient{
		bugzillaAddr:     addr,
		jsonRPCAddr:      u.String(),
		httpClient:       httpClient,
		seq:              0,
		m:                sync.RWMutex{},
		bugzillaLogin:    bugzillaLogin,
		bugzillaPassword: bugzillaPassword,
	}, nil
}

// Login allows to login using Bugzilla JSONRPC API, returns token
func (client *bugzillaJSONRPCClient) login() (err error) {
	klog.Infof("Authenticating to bugzilla via JSON")

	client.m.Lock()
	defer client.m.Unlock()

	args := make(map[string]interface{})
	args["login"] = client.bugzillaLogin
	args["password"] = client.bugzillaPassword
	args["remember"] = true

	var result map[string]interface{}
	err = client.call("User.login", &args, &result)
	if err != nil {
		return err
	}
	token, ok := result["token"].(string)
	if !ok {
		return fmt.Errorf("could not parse token %v", result["token"])
	}

	client.token = token

	return err
}

func (client *bugzillaJSONRPCClient) currentToken() string {
	client.m.RLock()
	defer client.m.RUnlock()
	return client.token
}

func (client *bugzillaJSONRPCClient) GetCookies() []*http.Cookie {
	url, _ := url.Parse(client.bugzillaAddr)
	cookies := client.httpClient.Jar.Cookies(url)
	return cookies
}

func (client *bugzillaJSONRPCClient) SetCookies(cookies []*http.Cookie) {
	url, _ := url.Parse(client.bugzillaAddr)
	client.httpClient.Jar.SetCookies(url, cookies)
}

// bugzillaVersion returns Bugzilla version
func (client *bugzillaJSONRPCClient) bugzillaVersion() (version string, err error) {
	var result map[string]interface{}
	err = client.call("Bugzilla.version", nil, &result)
	if err != nil {
		return "", err
	}
	version, ok := result["version"].(string)
	if !ok {
		return "", fmt.Errorf("could not parse token %v", result["version"])
	}
	return version, nil
}

// bugsInfo returns information about selected bugzilla tickets
func (client *bugzillaJSONRPCClient) bugsInfo(idList []int) (bugInfo map[string]interface{}, err error) {
	args := make(map[string]interface{})
	args["ids"] = idList
	args["token"] = client.currentToken()

	err = client.call("Bug.get", args, &bugInfo)
	if err != nil {
		return nil, err
	}
	return bugInfo, nil
}

// bugsHistory returns history of selected bugzilla tickets
func (client *bugzillaJSONRPCClient) bugsHistory(idList []int) (bugInfo map[string]interface{}, err error) {
	args := make(map[string]interface{})
	args["ids"] = idList
	args["token"] = client.currentToken()

	err = client.call("Bug.history", args, &bugInfo)
	if err != nil {
		return nil, err
	}
	return bugInfo, nil
}

// bugsHistory returns history of selected bugzilla tickets
func (client *bugzillaJSONRPCClient) addComment(id int, comment string) (commentInfo map[string]interface{}, err error) {
	args := make(map[string]interface{})
	args["id"] = id
	args["token"] = client.currentToken()
	args["comment"] = comment

	err = client.call("Bug.add_comment", args, &commentInfo)
	if err != nil {
		return nil, err
	}
	return commentInfo, nil
}

// call performs JSON RPC call
func (client *bugzillaJSONRPCClient) call(serviceMethod string, args interface{}, reply interface{}) error {
	var params [1]interface{}
	params[0] = args

	client.m.Lock()
	seq := client.seq
	client.seq++
	client.m.Unlock()

	cr := &clientRequest{
		Method: serviceMethod,
		Params: params,
		ID:     seq,
	}

	byteData, err := json.Marshal(cr)
	if err != nil {
		return err
	}

	res, err := client.authenticated(func() (*http.Response, error) {
		req, err := newHTTPRequest("POST", client.jsonRPCAddr, bytes.NewReader(byteData))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json-rpc")

		return client.httpClient.Do(req)
	})
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()
	if err != nil {
		return err
	}

	//body, _ := ioutil.ReadAll(res.Body)
	//fmt.Printf("%v", string(body))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Response status: %v", res.StatusCode)
	}

	v := &clientResponse{}

	err = json.NewDecoder(res.Body).Decode(v)
	if err != nil {
		return err
	}

	if v.Error != nil {
		return errors.New(v.Error.Message)
	}

	//fmt.Println(string(*v.Result))
	return json.Unmarshal(*v.Result, reply)
}

func (client *bugzillaJSONRPCClient) authenticated(f func() (*http.Response, error)) (*http.Response, error) {
	res, err := f()
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized {
		if err := client.login(); err != nil {
			return nil, err
		}
		res, err = f()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

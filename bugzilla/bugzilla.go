package bugzilla

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mangelajo/track/pkg/storecache"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/klog"
)

type Options struct {
	TrackConfigPath   string
	TrackDatabasePath string
}

func AddBugzillaFlags(opt *Options) {
	pflag.StringVar(&opt.TrackConfigPath, "bugzilla-track-config", "", "github.com/mangelajo/track .track config file path")
	pflag.StringVar(&opt.TrackDatabasePath, "bugzilla-track-database", "", "github.com/mangelajo/track track.db file path")
	pflag.String("bugzilla-url", "https://bugzilla.redhat.com", "Bugzilla URL")
	pflag.String("bugzilla-login", "", "Bugzilla login email")
	pflag.String("bugzilla-password", "", "Bugzilla login password")
	pflag.String("bugzilla-token", "", "Bugzilla API token, replacing login & password")

	viper.BindPFlag("bzurl", pflag.Lookup("bugzilla-url"))
	viper.BindEnv("bzurl", "BUGZILLA_URL")

	viper.BindPFlag("bzemail", pflag.Lookup("bugzilla-login"))
	viper.BindEnv("bzemail", "BUGZILLA_LOGIN")

	viper.BindPFlag("bzpass", pflag.Lookup("bugzilla-password"))
	viper.BindEnv("bzpass", "BUGZILLA_PASSWORD")

	viper.BindPFlag("bztoken", pflag.Lookup("bugzilla-token"))
	viper.BindEnv("bztoken", "BUGZILLA_TOKEN")
}

type Bugzilla struct {
	*Client
}

func NewBugzilla(opt Options) (*Bugzilla, error) {
	dir := opt.TrackDatabasePath

	if len(dir) == 0 {
		var err error
		dir, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}

	storePath := filepath.Join(dir, ".track.db")
	klog.Infof("Opening %s", storePath)
	storecache.Open(storePath)

	// load config
	if opt.TrackConfigPath != "" {
		viper.SetConfigFile(opt.TrackConfigPath)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		viper.AddConfigPath(homeDir)
		viper.SetConfigName(".track")
	}
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Could not read config file: %s \n", err)
	}

	// log in
	url := viper.GetString("bzurl")
	password := viper.GetString("bzpass")
	login := viper.GetString("bzemail")
	token := viper.GetString("bztoken")
	client, err := NewClient(url, login, password, token)
	if err != nil {
		return nil, err
	}

	return &Bugzilla{
		client,
	}, nil
}

func (bz *Bugzilla) Close() {
	storecache.Close()
}

func exampleTrackYaml() {
	fmt.Print(`
An example ~/.track.yaml:

bzurl: https://bugzilla.redhat.com
bzemail: xxxxx@redhat.com
bzpass: xxxxxxxx
dfg: Networking
htmlOpenCommand: xdg-open  # note: for OSX use open instead
queries:
    ovn-new: https://bugzilla.redhat.com/buglist.cgi?bug_status=NEW&classification=Red%20Hat&component=python-networking-ovn&list_id=8959835&product=Red%20Hat%20OpenStack&query_format=advanced
    ovn-rfes: https://bugzilla.redhat.com/buglist.cgi?bug_status=NEW&bug_status=ASSIGNED&bug_status=MODIFIED&bug_status=ON_DEV&bug_status=POST&bug_status=ON_QA&classification=Red%20Hat&component=python-networking-ovn&f1=keywords&f2=short_desc&j_top=OR&list_id=8959855&o1=substring&o2=substring&product=Red%20Hat%20OpenStack&query_format=advanced&v1=RFE&v2=RFE
users:
    colleague1@email.com
    colleague2@email.com

`)
}

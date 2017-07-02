package config

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("autotl")

func GetConfig() {

	// Transmission defaults
	viper.SetDefault("transmission_url", "http://127.0.0.1:9091")
	viper.SetDefault("transmission_user", "transmission")
	viper.SetDefault("transmission_password", "transmission")

	// if true it adds the torrents as paused
	viper.SetDefault("add_paused", false)

	// IRC defaults
	viper.SetDefault("irc_server", "irc.torrentleech.org:7021")
	viper.SetDefault("irc_channel", "#tlannounces")
	viper.SetDefault("irc_usetls", true)
	viper.SetDefault("irc_debug", false)
	viper.SetDefault("irc_verbose_handler", false)

	// Configuring and pulling overrides from environmental variables
	viper.SetEnvPrefix("AUTOTL")
	viper.AutomaticEnv()

	// Configuring the logger
	getLogger()

	// Password for the nick authentication
	if viper.GetString("irc_nickserv_password") == "" {
		displayHelp()
	}

	// Site username to generate the bot name "username_bot"
	if viper.GetString("username") == "" {
		displayHelp()
	}

	viper.Set("bot_name", fmt.Sprintf("%s_bot", viper.GetString("username")))

	// RSS key for link generation
	if viper.GetString("rss_key") == "" {
		displayHelp()
	}

	// No filters have been defined
	if len(viper.GetStringSlice("filters")) == 0 {
		displayHelp()
	}

	// Display parameters
	log.Info("Parameters:")
	for _, param := range []string{"transmission_url", "transmission_user",
		"transmission_password", "add_paused", "irc_server", "irc_channel",
		"irc_usetls", "irc_debug", "irc_verbose_handler",
		"irc_nickserv_password", "username", "bot_name", "rss_key",
	} {
		log.Infof("  %s = %s", param, viper.GetString(param))
	}

	// Display loaded fiters
	log.Info("Filters:")
	for _, filter := range viper.GetStringSlice("filters") {
		log.Infof("  - %s", filter)
	}
}

func displayHelp() {

	fmt.Println("You must have missed a few environmental variables!")
	fmt.Println("")
	fmt.Println("Make sure the following is defined:")
	fmt.Println("  - AUTOTL_IRC_NICKSERV_PASSWORD")
	fmt.Println("  - AUTOTL_USERNAME")
	fmt.Println("  - AUTOTL_RSS_KEY")
	fmt.Println("  - AUTOTL_FILTERS")
	fmt.Println("")

	os.Exit(1)
}

func getLogger() {

	// Nice and short log format
	formatter := logging.MustStringFormatter(
		`%{level:.1s} %{time:2006-01-02 15:04:05}  %{message}`)

	// Configure log backend
	logBackend := logging.NewLogBackend(os.Stdout, "", 0)

	// Bind log formats and log backend together
	backendFormatter := logging.NewBackendFormatter(logBackend, formatter)

	// Setup the log level and the backend
	backend := logging.AddModuleLevel(backendFormatter)
	backend.SetLevel(logging.INFO, "")

	logging.SetBackend(backend)
}

package main

import (
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"
	"time"

	"autotl/config"
	"autotl/irc"
	"autotl/transmission"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

const (
	releaseName    = "autotl"
	releaseVersion = "1.00"
	releaseDate    = "2017-07-02"
)

var log = logging.MustGetLogger("autotl")

// Transmission RPC documentation
// https://trac.transmissionbt.com/browser/trunk/extras/rpc-spec.txt

// getTorrentLeechLink function assembled a download link out using the
// torrentName, rssKey, and torrentID parameters
func getTorrentLeechLink(torrentID, rssKey, torrentName string) string {

	// Replacing spaces with dots
	dottedTorrentName := strings.Replace(torrentName, " ", ".", -1)

	// Returning with the formatted string
	return fmt.Sprintf("https://www.torrentleech.org/rss/download/%s/%s/%s.torrent",
		torrentID, rssKey, dottedTorrentName)
}

// alreadyAdded function downloads a list of currently added torrents
// from your transmission service and compares the names with the
// parameter of torrentName
func alreadyAdded(tc transmission.TransmissionClient, torrentName string) bool {

	dottedTorrentName := strings.Replace(torrentName, " ", ".", -1)

	torrents, err := tc.GetTorrents()

	// In case of a communication error returning with true so
	// further communication attempts with the service will be
	// avoided
	if err != nil {
		log.Error(err)
		return true
	}

	for _, torrent := range torrents {

		// fmt.Println("ID:", torrent.ID, "Name:", torrent.Name)
		if torrent.Name == dottedTorrentName {
			return true
		}
	}

	return false
}

// filterMatch function takes the defined regex filters and checks if the
// submitted torrentName parameter is a match or not. If it is a match
// it returns with true otherwise it returns with fals.
func filterMatch(torrentName string) bool {

	for _, filter := range viper.GetStringSlice("filters") {

		rex := regexp.MustCompile(filter)
		if rex.MatchString(torrentName) {
			return true
		}
	}

	return false
}

func main() {

	// Display bootstrap message
	fmt.Printf("%s v%s (%s)\n", releaseName, releaseVersion, releaseDate)
	fmt.Println("=====================================================")

	// Loading configuration
	config.GetConfig()

	// Connecting to the transmission API
	tc := transmission.New(
		viper.GetString("transmission_url"),
		viper.GetString("transmission_user"),
		viper.GetString("transmission_password"),
	)

	// Delimiter
	log.Info("--- --- --- --- --- --- --- --- ---")

	re := regexp.MustCompile(".*New Torrent Announcement:.* <([^>]*)>[\\W]*Name:'([^']*)[\\W]*uploaded by '([^']*)' -.*https://www.torrentleech.org/torrent/(\\d+).*$")

	//
	//  IRC
	//

	c := irc.IRC(viper.GetString("bot_name"), viper.GetString("username"))

	// Setting up some variables for the IRC connection
	c.VerboseCallbackHandler = viper.GetBool("irc_verbose_handler")
	c.Debug = viper.GetBool("irc_debug")
	c.UseTLS = viper.GetBool("irc_usetls")

	c.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	//
	//  IRC Handlers
	//

	c.AddCallback("001", func(e *irc.Event) {

		c.Join(viper.GetString("irc_channel"))
	})

	c.AddCallback("NOTICE", func(e *irc.Event) {

		// Looking out for messages from NickServ and authenticate nick if needed
		if e.Nick == "NickServ" {
			if strings.Contains(e.Arguments[1], "This nickname is registered and protected.") {
				log.Info("NickServ is asking for authentication. Username:", e.Arguments[0])
				log.Info("Sending password ...")
				log.Info("---")
				c.Privmsg(e.Nick, fmt.Sprintf("IDENTIFY %s", viper.GetString("irc_nickserv_password")))
			} else if strings.Contains(e.Arguments[1], "Password accepted - you are now recognized.") {
				log.Info("NickServ accepted your password!")
				log.Info("---")
			}
		}
	})

	c.AddCallback("PRIVMSG", func(e *irc.Event) {

		if e.Nick == "_AnnounceBot_" {

			log.Info("Announcement!")
			// fmt.Println("Message:", e.Arguments[1])
			sub := re.FindStringSubmatch(e.Arguments[1])

			// Parsing error?
			// It happens when the name of the torrent has an "'"
			if len(sub) != 5 {
				log.Error("  - parsing error!")
				log.Error("  - string:", e.Arguments[1])
				log.Error("--- --- --- --- --- --- --- --- ---")
				return
			}

			torrentCategory := sub[1]
			torrentName := sub[2]
			torrentID := sub[4]

			log.Info("Category:", torrentCategory)
			log.Info("Name:", torrentName)
			log.Info("ID:", torrentID)

			// Don't download anything which is french
			if strings.Contains(strings.ToLower(torrentName), "french") {
				log.Warning("  - FRENCH!")
				log.Warning("--- --- --- --- --- --- --- --- ---")
				return
			}

			// There is no match against any of the filters
			if !filterMatch(torrentName) {
				log.Warning("  - filtered out!")
				log.Warning("--- --- --- --- --- --- --- --- ---")
				return
			}

			// Transmission already has this torrent
			if alreadyAdded(tc, torrentName) {
				log.Warning("  - already added!")
				log.Warning("--- --- --- --- --- --- --- --- ---")
				return
			}

			// Generating download link
			torrentLink := getTorrentLeechLink(
				torrentID,
				viper.GetString("rss_key"),
				torrentName,
			)

			log.Info("  - link:", torrentLink)

			transmissionCommand, err := transmission.NewAddCmdByURL(torrentLink)

			if err != nil {
				log.Panic(err)
			}

			// Pausing it if AUTOTL_TRANSMISSION_ADD_PAUSED is set to true
			if viper.GetBool("add_paused") {
				transmissionCommand.Arguments.Paused = true
			}

			// Sleeping 5 seconds to give the tracker time to process the new torrent
			time.Sleep(5 * time.Second)

			// Injecting it into the transmission daemon through the API
			_, err = tc.ExecuteCommand(transmissionCommand)

			if err != nil {
				log.Panic(err)
			}

			log.Info("  - added!")
			log.Info("--- --- --- --- --- --- --- --- ---")
		}
	})

	err := c.Connect(viper.GetString("irc_server"))

	if err != nil {
		log.Errorf("Err %s", err)
		return
	}

	c.Loop()
}

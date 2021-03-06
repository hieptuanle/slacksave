package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

// You more than likely want your "Bot User OAuth Access Token" which starts with "xoxb-"

var accessToken = os.Getenv("SLACK_BOT_USER_ACCESS_TOKEN")
var verificationToken = os.Getenv("SLACK_BOT_VERIFICATION_TOKEN")
var userToken = os.Getenv("SLACK_USER_LEGACY_ACCESS_TOKEN")

var api = slack.New(accessToken)

var userAPI = slack.New(userToken)

func stringToTime(s string) (time.Time, error) {
	sec, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(sec), 0), nil
}

func getAllConversations(client *slack.Client, cursor string) (channels []slack.Channel, nextCursor string, err error) {
	return client.GetConversations(&slack.GetConversationsParameters{Limit: 20, Types: []string{"public_channel", "private_channel", "im", "mpim"}, Cursor: cursor})
}

func main() {

	userMap := make(map[string]slack.User)
	// var userMap map[string]slack.User = make

	users, err := api.GetUsers()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, user := range users {
		userMap[user.ID] = user
		fmt.Printf("ID: %s, Name: %s\n", user.ID, user.Name)
	}

	for groups, nextCursor, err := getAllConversations(userAPI, ""); len(groups) > 0; {
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
		fmt.Printf("Cursor %s\n", nextCursor)

		for _, group := range groups {
			if group.IsIM {
				if member, ok := userMap[group.User]; ok {
					fmt.Printf("Direct Group: %s\n", member.Name)
				}

			} else {
				fmt.Printf("Group Name: %s, Name: %s\n", group.ID, group.Name)
			}
		}

		if nextCursor != "" {
			groups, nextCursor, err = getAllConversations(userAPI, nextCursor)
		} else {
			groups = []slack.Channel{}
		}
	}

	fmt.Println(accessToken)
	fmt.Println(verificationToken)
	http.HandleFunc("/events-endpoint", func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.String()

		eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: verificationToken}))
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println("============================")
		fmt.Println("Type: " + eventsAPIEvent.Type)

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			case *slackevents.MessageEvent:
				var timestamp, e = stringToTime(ev.TimeStamp)
				if e != nil {
					fmt.Println(e)
					fmt.Println("Sai date")
					fmt.Println(timestamp)
				}

				fmt.Println("TS:" + timestamp.String())
				if ev.SubType == "message_changed" {
					fmt.Println("SubType:" + ev.SubType)
					fmt.Println("Text:" + ev.Message.Text)
				} else {
					fmt.Println("Text:" + ev.Text)
				}
				if ev.Username != "" {
					fmt.Println("Username:" + ev.Username)
				} else {
					fmt.Println("User:" + ev.User)
				}
				fmt.Println("Channel:" + ev.Channel)
			}
		}
	})
	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":8787", nil)
}

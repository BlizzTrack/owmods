package tools

import (
	"encoding/json"
	"github.com/markbates/goth"
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/database"
	"github.com/blizztrack/owmods/third_party/quill"
	"log"
	"strconv"
	"strings"
	"time"
)

func ImportOldGames() {
	if core.RedisManager.Exist("games-imported") {
		return
	}

	data, _ := readLines("./raw_importable/users.json")
	importOldUserList(data)

	/*
	data, _ = readLines("./raw_importable/games.json")
	importOldGamesList(data)
	*/

	log.Println("Imported all Overwatch Custom Games assets")

	core.RedisManager.Set("games-imported", "yes", 0)
}

func importOldUserList(lines []string) {
	for _, line := range lines {
		owGamesUsers := make(map[string]*json.RawMessage)
		err := json.Unmarshal([]byte(line), &owGamesUsers)
		if len(owGamesUsers) == 0 || err != nil {
			continue
		}

		if owGamesUsers["battletag"] == nil {
			continue
		}

		var bnet_id int
		json.Unmarshal(*owGamesUsers["bnet_id"], &bnet_id)

		var bnet_name string
		json.Unmarshal(*owGamesUsers["battletag"], &bnet_name)

		log.Printf("Importing User -> %s ( %d )", bnet_name, bnet_id)

		database.UpsertUser(goth.User{NickName: bnet_name, UserID: strconv.Itoa(bnet_id)})
	}
}

func importOldGamesList(lines []string) {
	for _, line := range lines {
		owGame := make(map[string]*json.RawMessage)
		json.Unmarshal([]byte(line), &owGame)

		var status string
		json.Unmarshal(*owGame["status"], &status)
		if strings.EqualFold(status, "deleted") {
			continue
		}

		var sharecode string
		json.Unmarshal(*owGame["sharecode"], &sharecode)

		var bnet_id int
		json.Unmarshal(*owGame["submitter_id"], &bnet_id)

		var title string
		json.Unmarshal(*owGame["title"], &title)

		var detail string
		json.Unmarshal(*owGame["description"], &detail)
		detail = strings.Replace(detail, "{\"ops\":", "", 1)
		detail = strings.TrimSuffix(detail, "}")
		html, err := quill.Render([]byte(detail))
		if err != nil {
			log.Printf("Failed game import -> %s", title)
			continue
		}

		var postedMap map[string]time.Time
		json.Unmarshal(*owGame["time"], &postedMap)
		posted := postedMap["$date"]

		var updateMap map[string]time.Time
		json.Unmarshal(*owGame["last_update"], &updateMap)
		updated := postedMap["$date"]

		var summary string
		json.Unmarshal(*owGame["summary"], &summary)

		log.Printf("Importing Game -> %s", title)

		if database.CodeInUseWorkshop(sharecode) {
			continue
		}

		database.ImportWorkshop(database.Workshop{
			Code:        sharecode,
			Title:       title,
			TLDR:        summary,
			Description: string(html),
			Author:      strconv.Itoa(bnet_id),
			Posted:      posted,
			Unix:        posted.UnixNano() / 1000000,
			Updated:     updated,
			UpdatedUnix: updated.UnixNano() / 1000000,
		})
	}
}


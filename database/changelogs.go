package database

import (
	"github.com/globalsign/mgo/bson"
	"github.com/blizztrack/owmods/core"
	"log"
	"time"
)

type ChangeLog struct {
	ID      bson.ObjectId `bson:"_id,omitempty" json:"-"`
	PostID  bson.ObjectId `bson:"_post"`
	Content string        `bson:"content"`
	Code    string        `bson:"code"`
	Version string        `bson:"version"`
	Posted  time.Time     `bson:"posted"`
	Unix    int64         `bson:"unix"`
}

const changeLogsTable = "change_logs"

func AddChangeLog(item ChangeLog) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(changeLogsTable)
	defer session.Close()

	item.ID = bson.NewObjectId()

	err = c.Insert(item)

	if err != nil {
		log.Printf("AddChangeLog Failed: %v", err)
	}
}

func UpdateChangeLog(content string, postID bson.ObjectId, unix int64) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(changeLogsTable)
	defer session.Close()

	err = c.Update(bson.M{
		"_post": postID,
		"unix":  unix,
	}, bson.M{"$set": bson.M{"content": content}})

	if err != nil {
		log.Printf("UpdateChangeLog Failed: %v", err)
	}
}

func ChangeLogExist(postID bson.ObjectId, unix int64) bool {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(changeLogsTable)
	defer session.Close()

	var out ChangeLog
	err = c.Find(bson.M{
		"_post": postID,
		"unix":  unix,
	}).Sort("-posted").One(&out)
	if err != nil {
		log.Printf("ChangeLogExist Failed: %v", err)
	}

	return out.Content != ""
}

func LatestChangeLog(postID bson.ObjectId) ChangeLog {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(changeLogsTable)
	defer session.Close()

	var out ChangeLog
	err = c.Find(bson.M{
		"_post": postID,
	}).Sort("-posted").One(&out)
	if err != nil {
		log.Printf("ChangeLogExist Failed: %v", err)
	}

	return out
}

func GetChangeLogs(postID bson.ObjectId, limit int) []ChangeLog {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(changeLogsTable)
	defer session.Close()

	var out []ChangeLog
	err = c.Find(bson.M{
		"_post": postID,
	}).Sort("-posted").Limit(limit).All(&out)

	return out
}

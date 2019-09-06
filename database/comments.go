package database

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/core/ts"
	"log"
	"time"
)

type Comment struct {
	ID      bson.ObjectId `bson:"_id,omitempty" json:"-"`
	PostID  bson.ObjectId `bson:"parent" json:"-"`
	Author  string        `bson:"author" form:"-"`
	Comment string        `bson:"comment" form:"comment"`
	Posted  time.Time     `bson:"posted" form:"-"`
	Unix    int64         `bson:"unix" form:"-"`
}

const commentTable = "comments"

func AddComment(comment Comment, parent bson.ObjectId) Comment {
	comment.ID = bson.NewObjectId()
	comment.PostID = parent
	comment.Posted, comment.Unix = ts.CurrentTimeToUnix()

	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(commentTable)
	defer session.Close()

	err = c.Insert(comment)
	if err != nil {
		log.Printf("AddComment Failed: %s", err)
		return Comment{}
	}

	return comment
}

func GetComments(parent bson.ObjectId, page, limit int) ([]Comment, int) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(commentTable)
	defer session.Close()

	page = page - 1
	if page < 0 {
		page = 0
	}

	query := bson.M{"parent": parent}
	offset := page * limit

	var comments []Comment
	err = c.Find(query).Sort("-posted").Skip(offset).Limit(limit).All(&comments)
	if err != nil {
		log.Printf("AddComment Failed: %s", err)
	}

	count, err := c.Find(query).Sort("-posted").Count()
	if err != nil {
		log.Println(err)
	}

	return comments, count
}

func DeleteComments(id bson.ObjectId) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(commentTable)
	defer session.Close()

	err = c.Remove(bson.M{"parent": id})
	if err != nil {
		fmt.Sprintf("Failed to remove comments: %v", err)
	}
}

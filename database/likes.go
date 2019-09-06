package database

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/core/ts"
	"log"
	"time"
)

type Like struct {
	ID        bson.ObjectId `bson:"_id,omitempty" json:"-"`
	PostID    bson.ObjectId `bson:"parent" json:"-"`
	Liker     string        `bson:"liker" json:"liker"`
	Liked     time.Time     `bson:"liked"`
	LikedUnix int64         `bson:"liked_unix"`
}

const (
	likesTable = "likes"
)

func AddLike(like Like) Like {
	like.ID = bson.NewObjectId()
	like.Liked, like.LikedUnix = ts.CurrentTimeToUnix()

	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(likesTable)
	defer session.Close()
	err = c.Insert(like)
	if err != nil {
		log.Printf("AddLike Failed: %s", err)
		return Like{}
	}

	return like
}

func AlreadyLikes(liker string, postID bson.ObjectId) bool {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(likesTable)
	defer session.Close()

	var like Like
	err = c.Find(bson.M{"parent": postID, "liker": liker}).One(&like)
	if err != nil {
		fmt.Sprintf("Failed to remove comments: %v", err)
	}

	return like.Liker == liker
}

func DeleteLike(liker string, postID bson.ObjectId) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(likesTable)
	defer session.Close()

	err = c.Remove(bson.M{"parent": postID, "liker": liker})
	if err != nil {
		fmt.Sprintf("Failed to remove comments: %v", err)
	}
}

func WorkshopCountLikes(id bson.ObjectId) int {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(likesTable)
	defer session.Close()

	count, err := c.Find(bson.M{"parent": id}).Count()
	if err != nil {
		fmt.Sprintf("Failed to remove comments: %v", err)
	}

	return count
}
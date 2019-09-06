package database

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/core/ts"
	"log"
	"strings"
	"time"
)

type Workshop struct {
	ID          bson.ObjectId `bson:"_id,omitempty" json:"-"`
	ShortID     string        `bson:"short_id" json:"-"`
	Code        string        `bson:"code" form:"code"`
	Title       string        `bson:"title" form:"title"`
	TLDR        string        `bson:"tldr" form:"tldr"`
	Description string        `bson:"desc" form:"desc"`
	Snippet     string        `bson:"snippet" form:"snippet"`
	Privacy     int           `bson:"privacy" form:"privacy"`
	Version     string        `bson:"version" form:"version"`
	Image       string        `bson:"image" form:"-"`
	ImagePath   string        `bson:"image_path" form:"-"`
	Mode        string        `bson:"-" form:"mode"`
	PostID      string        `bson:"-" form:"post_id"`
	ChangeLog   string        `bson:"-" form:"change_log"`
	Author      string        `bson:"author" form:"-"`
	Views       int64         `bson:"views" form:"-"`
	Score       float64       `bson:"score"`

	Posted      time.Time `bson:"posted" form:"-"`
	Unix        int64     `bson:"unix" form:"-"`
	Updated     time.Time `bson:"updated" form:"-"`
	UpdatedUnix int64     `bson:"updated_unix" form:"-"`

	// NoBSON
	Type string `bson:"-" form:"type"`
}

const (
	workshopTable = "workshop"
	WorkshopLimit = 24
)

func UpdateWorkshop(item Workshop) Workshop {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	if item.ShortID == "" {
		// Force update the legacy to the new system
		item.ShortID = core.ShortID().MustGenerate()
	}

	err = c.Update(bson.M{"_id": item.ID}, item)
	if err != nil {
		log.Printf("UpdateWorkshop Failed: %v", err)
		return item
	}

	return item
}

func CodeInUseWorkshop(code string) bool {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	var item Workshop
	err = c.Find(bson.M{"code": &bson.RegEx{
		Pattern: code,
		Options: "i",
	}}).One(&item)
	if err != nil {
		log.Printf("CodeExistWorkshop failed: %v", err)
	}

	return strings.EqualFold(item.Code, code)
}

func SingleWorkshop(id string) Workshop {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	var item Workshop
	if bson.IsObjectIdHex(id) {
		err = c.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&item)
		if err != nil {
			log.Printf("SingleWorkshop failed: %v", err)
			return Workshop{}
		}
	} else {
		err = c.Find(bson.M{"$or": []bson.M{
			{"short_id": id},
			{"code": id},
		}}).One(&item)
		if err != nil {
			log.Printf("GetUser Failed: %v", err)
			return Workshop{}
		}
	}
	return item
}

func AddWorkshop(item Workshop) Workshop {
	i := bson.NewObjectId()

	item.ID = i
	item.Posted, item.Unix = ts.CurrentTimeToUnix()
	item.ShortID = core.ShortID().MustGenerate()

	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	err = c.Insert(item)
	if err != nil {
		log.Printf("AddWorkshop failed: %v", err)
		return Workshop{}
	}

	return item
}

func ImportWorkshop(item Workshop) Workshop {
	i := bson.NewObjectId()

	item.ID = i
	item.ShortID = core.ShortID().MustGenerate()

	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	err = c.Insert(item)
	if err != nil {
		log.Printf("AddWorkshop failed: %v", err)
		return Workshop{}
	}

	return item
}

func SearchWorkShop(query bson.M, page, limit int, sort string) ([]Workshop, int) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	sort = strings.ToLower(sort)
	if sort == "" {
		sort = "-posted"
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	page = page - 1
	if page < 0 {
		page = 0
	}

	offset := page * limit
	searchQuery := query

	q := c.Find(searchQuery)
	t := c.Find(searchQuery)

	var returns []Workshop
	err = q.Sort(sort).Skip(offset).Limit(limit).All(&returns)
	if err != nil {
		log.Println(err)
	}

	count, err := t.Sort().Count()
	if err != nil {
		log.Println(err)
	}

	return returns, count
}

func DeleteWorkshop(id bson.ObjectId) {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	err = c.Remove(bson.M{"_id": id})
	if err != nil {
		fmt.Sprintf("Failed to remove: %v", err)
	}
}

func RandomWorkshop() Workshop {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	var result Workshop
	err = c.Pipe([]bson.M{
		{"$match": bson.M{"privacy": 0}},
		{"$sample": bson.M{"size": 5}},
	}).One(&result)
	if err != nil {
		fmt.Printf("Failed to get random: %v", err)
	}

	return result
}

func WorkshopCreatorCount(creator string) int {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	count, _ := c.Find(bson.M{
		"author": creator,
	}).Count()

	return count
}

func (w Workshop) UpdateViews() {
	//  $inc:

	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(workshopTable)
	defer session.Close()

	err = c.Update(bson.M{"_id": w.ID}, bson.M{
		"$inc": bson.M{"views": 1},
	})
	if err != nil {
		log.Printf("UpdateViews Failed: %v", err)
	}
}

func (w Workshop) GetID() string {
	if w.ShortID == "" {
		return w.ID.Hex()
	}

	return w.ShortID
}

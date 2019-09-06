package routes

import (
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/kataras/iris"
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/database"
	"github.com/blizztrack/owmods/system"
	"strings"
	"time"
)

type viewingRoutes struct{}

func NewViewingRoutes(party iris.Party) {
	v := viewingRoutes{}

	party.Get("/random", v.viewRandomGame)
	party.Get("/{id:string}", v.viewingGame)
	party.Post("/{id:string}/comment", isLoggedIn, v.postComment)
	party.Get("/{id:string}/comment", v.getComments)
	party.Post("/{id:string}/likes", v.postLike)
	party.Get("/{id:string}/likes", v.getLike)

}

func (viewingRoutes) viewRandomGame(ctx iris.Context) {
	item := database.RandomWorkshop()
	if item.Code == "" {
		ctx.Redirect("/")
		return
	}

	ctx.Redirect(fmt.Sprintf("/%s", item.GetID()))
}

func (viewingRoutes) viewingGame(ctx iris.Context) {
	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.NotFound()
		return
	}

	sessAuthor := ""
	sess := system.Session().Get(ctx)
	val := sess.Get("User")
	if val != nil {
		sessAuthor = val.(map[string]interface{})["BattleID"].(string)
	}

	if sessAuthor == "" {
		if item.Privacy == 1 {
			ctx.NotFound()
			return
		}
	} else {
		if item.Author != sessAuthor && item.Privacy == 1 {
			ctx.NotFound()
			return
		}
	}

	lastViewed := system.Session().Get(ctx).GetStringDefault("last_viewed", "")
	if lastViewed != item.ID.Hex() {
		system.Session().Get(ctx).Set("last_viewed", item.ID.Hex())
	} else {
		item.UpdateViews()
	}

	author := database.GetUser(item.Author)

	authorName := author.GetNickNameNoTag()
	if author.Name != "" {
		authorName = author.Name
	}
	ctx.ViewData("Header", core.Render.Header(
		fmt.Sprintf("%s by, %s", item.Title, authorName),
		"bt",
		item.TLDR,
	))

	changeItems := database.GetChangeLogs(item.ID, 6)
	ctx.ViewData("Workshop", item)
	ctx.ViewData("Author", author)
	ctx.ViewData("ICreated", item.Author == sessAuthor)
	if len(changeItems) > 0 {
		ctx.ViewData("ChangeLog", changeItems)
		ctx.ViewData("ChangeLogCount", int32(len(changeItems)-1))
	} else {
		ctx.ViewData("ChangeLog", false)
	}

	urlImage := item.Image
	if urlImage != "" {
		ctx.ViewData("UserImage", urlImage)
		ctx.ViewData("UseLargeImage", true)
	}

	ctx.View("view_workshop.tmpl")
}

func (viewingRoutes) postComment(ctx iris.Context) {
	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.JSON(map[string]string{
			"error": "not found",
			"name":  "system", // TODO
		})
		return
	}

	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	author := data["BattleID"].(string)

	var post database.Comment
	err := ctx.ReadForm(&post)
	if err != nil {
		ctx.WriteString("error:Internal server error")
		return
	}

	post.Comment = strings.TrimSpace(strip.StripTags(post.Comment))
	if len(post.Comment) < 5 || len(post.Comment) > 500 {
		ctx.WriteString("error:Comment must be between 5 and 500 characters")
		return
	}

	key := fmt.Sprintf("%s-%s-comment", item.ID.Hex(), author)
	if core.RedisManager.Exist(key) {
		ctx.WriteString("")
		return
	}
	core.RedisManager.Set(key, "rate-limited", 1*time.Minute)

	post.Author = author
	post = database.AddComment(post, item.ID)

	user := database.GetUser(author)
	ctx.ViewLayout(iris.NoLayout)
	ctx.ViewData("Comments", []database.Comment{post})
	ctx.ViewData("Authors", map[string]database.User{author: user})
	ctx.View("read_comment.tmpl")
}

func (viewingRoutes) getComments(ctx iris.Context) {
	page := ctx.URLParamIntDefault("page", 1)

	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.JSON(map[string]string{
			"error": "not found",
			"name":  "system", // TODO
		})
		return
	}

	comments, _ := database.GetComments(item.ID, page, 10)

	users := make([]string, 0)
	for _, comment := range comments {
		users = appendIfMissing(users, comment.Author)
	}

	mapUser := make(map[string]database.User)
	for _, user := range database.BulkFindUser(users...) {
		mapUser[user.BattleID] = user
	}

	ctx.ViewLayout(iris.NoLayout)
	ctx.ViewData("Comments", comments)
	ctx.ViewData("Authors", mapUser)
	ctx.View("read_comment.tmpl")
}

func (viewingRoutes) postLike(ctx iris.Context) {
	val := system.Session().Get(ctx).Get("User")
	if val == nil {
		ctx.JSON(map[string]string{
			"error": "must be logged in to use this action",
			"name":  "system", // TODO
		})
		return
	}

	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.JSON(map[string]string{
			"error": "not found",
			"name":  "system", // TODO
		})
		return
	}

	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	author := data["BattleID"].(string)

	like := database.Like{
		PostID:    item.ID,
		Liker:     author,
	}

	if database.AlreadyLikes(like.Liker, like.PostID) {
		database.DeleteLike(like.Liker, like.PostID)
		count := database.WorkshopCountLikes(like.PostID)

		ctx.JSON(map[string]interface{}{
			"ok": "workshop has been unliked",
			"deleted": true,
			"count": count,
		})
		return
	}

	database.AddLike(like)
	count := database.WorkshopCountLikes(like.PostID)
	ctx.JSON(map[string]interface{}{
		"ok": "workshop has been liked",
		"deleted": false,
		"count": count,
	})
}

func (viewingRoutes) getLike(ctx iris.Context) {
	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.JSON(map[string]string{
			"error": "not found",
			"name":  "system", // TODO
		})
		return
	}

	count := database.WorkshopCountLikes(item.ID)
	ctx.JSON(map[string]interface{}{
		"count": count,
	})
}

func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

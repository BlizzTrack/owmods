package routes

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/blang/semver"
	"github.com/globalsign/mgo/bson"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/kataras/iris"
	"github.com/microcosm-cc/bluemonday"
	wordfilter "github.com/syyongx/go-wordsfilter"
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/core/paging"
	"github.com/blizztrack/owmods/core/ts"
	"github.com/blizztrack/owmods/database"
	"github.com/blizztrack/owmods/system"
	"github.com/blizztrack/owmods/web_server/helpers"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type userRoutes struct{}

var p = bluemonday.UGCPolicy()

const Description = "A simple website that will let you share your workshop creations!"

var (
	isValidGameCode = regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString
	isValidURL      = regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString
	allowedTypes    = []string{
		"image/jpeg",
		"image/png",
		"image/gif",
	}
	fileSizeLimit        = 2621440
	fileProfileSizeLimit = []int{
		512, 512,
	}

	wordList     = []string{}
	wordListUser = []string{}

	wf          *wordfilter.WordsFilter
	wfRoot      map[string]*wordfilter.Node
	wfRootUsers map[string]*wordfilter.Node
)

func isLoggedIn(ctx iris.Context) {
	val := system.Session().Get(ctx).Get("User")
	if val == nil {
		ctx.Redirect("/")
		return
	}

	ctx.Next()
}

func init() {
	file, err := ioutil.ReadFile("./assets/json/en_out.json")
	if err != nil {
		log.Panicln(err)
	}

	json.Unmarshal(file, &wordList)
	json.Unmarshal(file, &wordListUser)
	wordListUser = append(wordListUser, "settings")
	wordListUser = append(wordListUser, "tools")

	wf = wordfilter.New()
	wfRoot = wf.Generate(wordList)
	wfRootUsers = wf.Generate(wordListUser)
}

func NewUserRoutes(party iris.Party) {
	u := userRoutes{}

	helpers.AllowYoutube(p)

	party.Get("/tools/add_workshop", isLoggedIn, u.userAddWorkshop)
	party.Post("/tools/add_workshop", isLoggedIn, u.userAddWorkshopPost)

	party.Get("/tools/update_workshop/{id:string}", isLoggedIn, u.userUpdateWorkshop)
	party.Post("/tools/update_workshop/{id:string}", isLoggedIn, u.userAddWorkshopPost)
	party.Post("/tools/nuke_workshop/{id:string}", isLoggedIn, u.userNukeWorkshopPost)

	party.Get("/settings", isLoggedIn, u.userAccountSettings)
	party.Post("/settings", isLoggedIn, u.userUpdateAccountSettings)
	party.Get("/{id:string}/icon", u.userGetIcon)
	party.Get("/{id:string}", u.userViewWList)
}

func (userRoutes) userGetIcon(ctx iris.Context) {
	user := database.GetUser(ctx.Params().GetStringDefault("id", ""))
	if user.BattleID == "" {
		ctx.ServeFile("./assets/static/images/bt.png", true)
		return
	}
	key := fmt.Sprintf("%s-icon", user.BattleID)
	if core.RedisManager.Exist(key) {
		value, _ := core.RedisManager.Get(key)
		ctx.Redirect(value)
		return
	}

	res, err := http.Get("https://playoverwatch.com/en-us/career/pc/" + strings.Replace(user.NickName, "#", "-", 1))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".masthead-player").Each(func(i int, s *goquery.Selection) {
		img, _ := s.Find("img.player-portrait").Attr("src")

		core.RedisManager.Set(key, img, 5*time.Minute)
		ctx.Redirect(img)
	})
}

func (userRoutes) userAccountSettings(ctx iris.Context) {
	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	loggedin := data["BattleID"].(string)
	user := database.GetUser(loggedin)

	ctx.ViewData("User", user)
	ctx.ViewData("Header", core.Render.Header(
		"Editing account settings",
		"bt",
		Description,
	))

	ctx.View("user_edit_profile.tmpl")
}

func (userRoutes) userUpdateAccountSettings(ctx iris.Context) {
	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	loggedin := data["BattleID"].(string)
	user := database.GetUser(loggedin)
	currentURL := user.URL
	currentName := user.Name

	ctx.ReadForm(&user)

	if user.Mode == "raw" {
		switch user.Type {
		case "image":
			if user.Image != "" {
				core.AWSClient().DeleteFile(user.ImagePath)
				user.Image = ""
				user.ImagePath = ""
				database.UpdateUser(user)
			}
			ctx.JSON(map[string]string{
				"ok": "image has been reset",
			})
			return
		}
	}

	if wf.Contains(user.Name, wfRoot) {
		ctx.JSON(map[string]string{
			"error": "name contains banned words please try again",
			"name":  "name",
		})
		return
	}

	if wf.Contains(user.URL, wfRootUsers) {
		ctx.JSON(map[string]string{
			"error": "url contains banned words please try again",
			"name":  "url",
		})
		return
	}

	mimeType := ""
	file, header, _ := ctx.FormFile("image")
	if header != nil {
		fileHeader := make([]byte, 512)
		if _, err := file.Read(fileHeader); err != nil {
			log.Println(err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			log.Println(err)
		}

		mimeType = http.DetectContentType(fileHeader)
		if header.Size > int64(fileSizeLimit) {
			ctx.JSON(map[string]string{
				"error": "file can be a max of 2.5mb",
				"name":  "code",
			})
			return
		}

		if !stringInSlice(strings.ToLower(mimeType), allowedTypes) {
			ctx.JSON(map[string]string{
				"error": "only JPG, PNG, and Gif are supported",
				"name":  "code",
			})
			return
		}
	}

	if user.Name != currentName {
		if len(user.Name) > 0 && currentName != "" {
			ctx.JSON(map[string]string{
				"error": "cannot change user name",
				"name":  "name",
			})
			return
		}
	}

	if user.URL != currentURL {
		if len(user.URL) > 0 && currentURL != "" {
			ctx.JSON(map[string]string{
				"error": "cannot change url",
				"name":  "name",
			})
			return
		}
	}

	if len(user.URL) > 0 {
		// Check if they created stuff
		count := database.WorkshopCreatorCount(loggedin)
		if count < 2 {
			ctx.JSON(map[string]string{
				"error": "you are now allowed to have a custom URL",
				"name":  "name",
			})
			return
		}
	}

	user.Name = strings.TrimSpace(strip.StripTags(user.Name))
	if len(user.Name) < 3 || len(user.Name) > 20 || !isValidGameCode(user.Name) {
		ctx.JSON(map[string]string{
			"error": "name must be between 3 and 20 characters",
			"name":  "name",
		})
		return
	}
	user.NameLower = strings.ToLower(user.Name)

	user.URL = strings.TrimSpace(strip.StripTags(user.URL))
	if len(user.URL) < 3 || len(user.URL) > 20 || !isValidURL(user.URL) {
		ctx.JSON(map[string]string{
			"error": "url must be between 3 and 20 characters",
			"name":  "url",
		})
		return
	}

	if user.URL != currentURL {
		exist := database.GetUser(user.URL)
		if exist.Name != "" {
			ctx.JSON(map[string]string{
				"error": "url already in use try another",
				"name":  "url",
			})
			return
		}
	}

	if header != nil {
		w := fileProfileSizeLimit[0]
		h := fileProfileSizeLimit[1]
		rW, rH := core.AWSClient().GetImageSize(file)
		if rW != rH || rW > w || rH > h {
			ctx.JSON(map[string]string{
				"error": "image must be square and between 512x512",
				"name":  "image",
			})
			return
		}

		path := core.AWSClient().CreateFileName(filepath.Ext(header.Filename))
		_, err := core.AWSClient().PutFile(path, mimeType, header.Size, file)
		if err != nil {
			log.Fatal(err)
		}

		core.AWSClient().DeleteFile(user.ImagePath)
		user.Image = fmt.Sprintf("%s%s", core.AWSClient().BaseURL(), path)
		user.ImagePath = path
	}

	database.UpdateUser(user)

	ctx.JSON(map[string]string{
		"ok": "updated account settings",
	})
	// log.Printf("%+v", user)
}

func (userRoutes) userViewWList(ctx iris.Context) {
	page := ctx.URLParamIntDefault("page", 1)
	user := database.GetUser(ctx.Params().GetStringDefault("id", ""))
	sort := ctx.URLParamDefault("sort", "1")
	switch sort {
	case "1":
		sort = "-posted"
		break
	case "2":
		sort = "+posted"
		break
	case "3":
		sort = "-updated"
		break
	default:
		sort = "-posted"
		break
	}

	if user.BattleID == "" {
		ctx.NotFound()
		return
	}

	query := ctx.URLParamDefault("search", "")
	searchQuery := bson.M{
		"author": user.BattleID,
	}
	val := system.Session().Get(ctx).Get("User")
	if val == nil {
		searchQuery["privacy"] = bson.M{
			"$in": []interface{}{nil, 0},
		}
	} else {
		v := val.(map[string]interface{})
		if user.BattleID != v["BattleID"].(string) {
			searchQuery["privacy"] = bson.M{
				"$in": []interface{}{nil, 0},
			}
		}
	}

	if query != "" {
		searchQuery["$or"] = []bson.M{
			{"title": &bson.RegEx{
				Pattern: query,
				Options: "i",
			},},
			{"tldr": &bson.RegEx{
				Pattern: query,
				Options: "i",
			},},
		}
	}

	workShops, total := database.SearchWorkShop(searchQuery, page, database.WorkshopLimit, sort)

	mapUser := map[string]database.User{
		user.BattleID: user,
	}

	ctx.ViewData("Workshops", workShops)
	ctx.ViewData("Authors", mapUser)

	urlImage := user.Image
	if urlImage == "" {
		urlImage = fmt.Sprintf("https://owmods.com/u/%s/icon", user.GetID())
	}
	ctx.ViewData("UserImage", urlImage)

	authorName := user.GetNickNameNoTag()
	if user.Name != "" {
		authorName = user.Name
	}
	ctx.ViewData("Header", core.Render.Header(
		fmt.Sprintf("Workshops by, %s", authorName),
		"bt",
		Description,
	))

	url := user.ID.Hex()
	if user.URL != "" {
		url = user.URL
	}

	urlQuery := ctx.URLParams()
	if _, ok := urlQuery["page"]; !ok {
		urlQuery["page"] = strconv.Itoa(page)
	}

	pager := paging.New(total, database.WorkshopLimit, page, fmt.Sprintf("/u/%s?%s", url, helpers.ToURLQuery(urlQuery).Encode()))

	ctx.ViewData("Pager", pager)
	ctx.ViewData("Total", len(workShops))
	ctx.ViewData("sort", sort)

	ctx.View("index.tmpl")
}

func (userRoutes) userAddWorkshop(ctx iris.Context) {
	ctx.ViewData("Header", core.Render.Header(
		"Adding new Workshop",
		"bt",
		Description,
	))

	ctx.View("add_workshop.tmpl")
}

func (userRoutes) userUpdateWorkshop(ctx iris.Context) {
	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.NotFound()
		return
	}

	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	author := data["BattleID"].(string)
	if item.Author != author {
		ctx.NotFound()
		return
	}

	ctx.ViewData("Header", core.Render.Header(
		fmt.Sprintf("Updating %s", item.Title),
		"bt",
		Description,
	))

	ctx.ViewData("Workshop", item)
	ctx.ViewData("Changelog", database.LatestChangeLog(item.ID))
	ctx.ViewData("Mode", "update")
	ctx.ViewData("Privacy", strconv.Itoa(item.Privacy))

	ctx.View("add_workshop.tmpl")
}

func (userRoutes) userAddWorkshopPost(ctx iris.Context) {
	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	author := data["BattleID"].(string)

	var post database.Workshop
	_ = ctx.ReadForm(&post)

	mimeType := ""

	post.Code = strings.ToUpper(post.Code)
	post.Title = strings.TrimSpace(strip.StripTags(post.Title))
	post.TLDR = strings.TrimSpace(strip.StripTags(post.TLDR))
	post.Description = strings.TrimSpace(p.Sanitize(post.Description))
	post.Code = strings.TrimSpace(strip.StripTags(post.Code))
	post.ChangeLog = strings.TrimSpace(p.Sanitize(post.ChangeLog))

	file, header, _ := ctx.FormFile("image")
	if header != nil {
		fileHeader := make([]byte, 512)
		if _, err := file.Read(fileHeader); err != nil {
			log.Println(err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			log.Println(err)
		}

		mimeType = http.DetectContentType(fileHeader)

		log.Println(mimeType)

		log.Println(header.Size)
		if header.Size > int64(fileSizeLimit) {
			ctx.JSON(map[string]string{
				"error": "file can be a max of 2.5mb",
				"name":  "code",
			})
			return
		}

		if !stringInSlice(strings.ToLower(mimeType), allowedTypes) {
			ctx.JSON(map[string]string{
				"error": "only JPG, PNG, and Gif are supported",
				"name":  "code",
			})
			return
		}
	}

	exist := database.CodeInUseWorkshop(post.Code)

	id := ctx.Params().GetStringDefault("id", "")
	var item database.Workshop
	if id != "" {
		item = database.SingleWorkshop(id)
	}

	if post.Mode == "raw" {
		if item.Author != author {
			ctx.JSON(map[string]string{
				"error": "you did not create the workshop",
			})
			return
		}

		switch post.Type {
		case "image":
			if item.Image != "" {
				core.AWSClient().DeleteFile(item.ImagePath)
				item.Image = ""
				item.ImagePath = ""
				database.UpdateWorkshop(item)
			}
			ctx.JSON(map[string]string{
				"ok": "image has been reset",
			})
			return
		}
	}

	if post.Privacy > 2 || post.Privacy < 0 {
		post.Privacy = 0
	}

	if wf.Contains(post.Title, wfRoot) {
		ctx.JSON(map[string]string{
			"error": "title contains banned words please try again",
			"name":  "title",
		})
		return
	}

	if !isValidGameCode(post.Code) {
		ctx.JSON(map[string]string{
			"error": "this game code is invalid",
			"name":  "code",
		})
		return
	}

	if exist {
		if item.Code == "" || !strings.EqualFold(item.Code, post.Code) {
			ctx.JSON(map[string]string{
				"error": "code already in use",
				"name":  "code",
			})
			return
		}
	}

	if _, err := semver.Make(post.Version); err != nil {
		ctx.JSON(map[string]string{
			"error": "version must follow semantic versioning (ex: 1.0.0)",
			"name":  "version",
		})
		return
	}

	if len(post.Code) < 5 || len(post.Code) > 10 {
		ctx.JSON(map[string]string{
			"error": "code must be between 5 and 10 characters",
			"name":  "code",
		})
		return
	}

	if len(post.TLDR) < 15 || len(post.TLDR) > 150 {
		ctx.JSON(map[string]string{
			"error": "TLDR must be between 15 and 150 characters",
			"name":  "tldr",
		})
		return
	}

	if len(post.Title) < 3 || len(post.Title) > 100 {
		ctx.JSON(map[string]string{
			"error": "title must be between 3 and 100 characters",
			"name":  "title",
		})
		return
	}

	if len(post.Description) < 15 || len(post.Description) > 150000 {
		ctx.JSON(map[string]string{
			"error": "description must be between 15 and 150,000 characters",
			"name":  "desc",
		})
		return
	}

	// Owner Validation
	if strings.EqualFold(post.Mode, "update") {
		if len(post.ChangeLog) > 1000 {
			ctx.JSON(map[string]string{
				"error": "change log can only be a max of 1,000 characters",
				"name":  "change_log",
			})
			return
		}

		// todo
		if item.Code == "" {
			ctx.NotFound()
			return
		}

		if item.Author != author {
			ctx.JSON(map[string]string{
				"error": "you did not create the workshop",
			})
			return
		}

		post.Code = strip.StripTags(post.Code)
		if post.Code != item.Code {
			/*
				item.Updated = time.Now()
				item.UpdatedUnix = item.Updated.UnixNano() / 1000000
			*/
			item.Updated, item.Unix = ts.CurrentTimeToUnix()
			if len(post.ChangeLog) > 0 {
				database.AddChangeLog(database.ChangeLog{
					PostID:  item.ID,
					Content: post.ChangeLog,
					Posted:  item.Updated,
					Unix:    item.UpdatedUnix,
					Code:    post.Code,
					Version: post.Version,
				})
			}
		} else {
			if len(post.ChangeLog) > 0 {
				var updateTime int64
				var update time.Time
				if item.UpdatedUnix == 0 {
					updateTime = item.Unix
					update = item.Posted
				} else {
					updateTime = item.UpdatedUnix
					update = item.Updated
				}

				if database.ChangeLogExist(item.ID, updateTime) {
					database.UpdateChangeLog(post.ChangeLog, item.ID, updateTime)
				} else {
					database.AddChangeLog(database.ChangeLog{
						PostID:  item.ID,
						Content: post.ChangeLog,
						Posted:  update,
						Unix:    updateTime,
						Code:    post.Code,
						Version: post.Version,
					})
				}
			}
		}

		item.Title = strip.StripTags(post.Title)
		item.TLDR = strip.StripTags(post.TLDR)
		item.Description = p.Sanitize(post.Description)
		item.Code = post.Code
		item.Privacy = post.Privacy
		item.Version = post.Version
		if header != nil {
			path := core.AWSClient().CreateFileName(filepath.Ext(header.Filename))
			_, err := core.AWSClient().PutFile(path, mimeType, header.Size, file)
			if err != nil {
				log.Fatal(err)
			}

			core.AWSClient().DeleteFile(item.ImagePath)
			item.Image = fmt.Sprintf("%s%s", core.AWSClient().BaseURL(), path)
			item.ImagePath = path
		}

		item = database.UpdateWorkshop(item)

		ctx.JSON(map[string]string{
			"ok": "updated workshop",
		})
		return
	}

	post.Author = author
	if header != nil {
		path := core.AWSClient().CreateFileName(filepath.Ext(header.Filename))
		_, err := core.AWSClient().PutFile(path, mimeType, header.Size, file)
		if err != nil {
			log.Fatal(err)
		}

		post.Image = fmt.Sprintf("%s%s", core.AWSClient().BaseURL(), path)
		post.ImagePath = path
	}

	post = database.AddWorkshop(post)

	ctx.JSON(map[string]string{
		"ok": fmt.Sprintf("url:%s", post.GetID()),
	})
}

func (userRoutes) userNukeWorkshopPost(ctx iris.Context) {
	data := system.Session().Get(ctx).Get("User").(map[string]interface{})
	author := data["BattleID"].(string)

	item := database.SingleWorkshop(ctx.Params().GetStringDefault("id", ""))
	if item.Code == "" {
		ctx.JSON(map[string]string{
			"error": "not found",
			"name":  "system", // TODO
		})
		return
	}

	if author != item.Author {
		ctx.JSON(map[string]string{
			"error": "not found",
			"name":  "system", // TODO
		})
		return
	}

	database.DeleteWorkshop(item.ID)
	database.DeleteComments(item.ID)
	if item.ImagePath != "" {
		core.AWSClient().DeleteFile(item.ImagePath)
	}

	ctx.JSON(map[string]string{
		"ok": "deleted the workshop",
	})
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

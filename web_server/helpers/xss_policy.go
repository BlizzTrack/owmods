package helpers

import (
	"github.com/microcosm-cc/bluemonday"
	"log"
	"regexp"
)

var regex = `((http:\/\/(gfycat\.com\/.*|.*youtube\.com\/watch.*|.*\.youtube\.com\/v\/.*|youtu\.be\/.*|.*\.youtube\.com\/user\/.*|.*\.youtube\.com\/.*#.*\/.*|m\.youtube\.com\/watch.*|m\.youtube\.com\/index.*|.*\.youtube\.com\/profile.*|.*\.youtube\.com\/view_play_list.*|.*\.youtube\.com\/playlist.*|www\.youtube\.com\/embed\/.*|youtube\.com\/gif.*|www\.youtube\.com\/gif.*|www\.youtube\.com\/attribution_link.*|youtube\.com\/attribution_link.*|youtube\.ca\/.*|youtube\.jp\/.*|youtube\.com\.br\/.*|youtube\.co\.uk\/.*|youtube\.nl\/.*|youtube\.pl\/.*|youtube\.es\/.*|youtube\.ie\/.*|it\.youtube\.com\/.*|youtube\.fr\/.*|.*\.twitch\.tv\/.*|twitch\.tv\/.*))|(https:\/\/(gfycat\.com\/.*|.*youtube\.com\/watch.*|.*\.youtube\.com\/v\/.*|youtu\.be\/.*|.*\.youtube\.com\/playlist.*|www\.youtube\.com\/embed\/.*|youtube\.com\/gif.*|www\.youtube\.com\/gif.*|www\.youtube\.com\/attribution_link.*|youtube\.com\/attribution_link.*|youtube\.ca\/.*|youtube\.jp\/.*|youtube\.com\.br\/.*|youtube\.co\.uk\/.*|youtube\.nl\/.*|youtube\.pl\/.*|youtube\.es\/.*|youtube\.ie\/.*|it\.youtube\.com\/.*|youtube\.fr\/.*|.*\.twitch\.tv\/.*|twitch\.tv\/.*)))`
func AllowYoutube(p *bluemonday.Policy) {
	r, err := regexp.Compile(regex)
	if err != nil {
		log.Panicf("Regex youtube failed: %v", err)
	}

	p.AllowAttrs("height", "width").Matching(bluemonday.NumberOrPercent).OnElements("iframe")
	p.AllowAttrs("src").Matching(r).OnElements("iframe")
}

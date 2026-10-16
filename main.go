/*
Object Hub openGo
Contributors:
MIOBOMB - architecture + initial work + fun work
DenisC - Entire search operation algorithm
//
Author: MIOBOMB (2026)
ojhub-opengo - open implementation of Object Hub backend in Go(lang)
Since php/node backends of this site run on freebsd we deliberately
abandoned dockers and other infrastructure for the purpose of total portability
of this backend implementation, because our freebsd server cannot run Docker,
and many "fashionable" technologies from the linux world, such as the cursed SystemD.
And personally, I am MIOBOMB, and I have FreeBSD on my desktop,
you probably have Windows, and someone else hasn't set up WSL2 yet.
//
PROJECT GOAL FOR THE NEXT 2 MONTHS:
Replacing the hottest sections with Golang, and getting rid of the nodejs part of the backend
with returning php to those places that go cannot serve for some time
//
LONG-TERM PROJECT GOAL:
openGo aims for 100% compatibility and replaceability of the php backend,
to the point that routes, input and output data will not change at all.
Note that for such an implementation of endpoints you will have to understand
the client's source code! openGo was based on version 0.97.33 and 0.97.7,
because that way it will be possible to recreate 100% emulation of the php backend.
//
FIXME:
think about integrating nginx and redis into the openGo binary
when openGo is close to 100% readiness

CODE AND ARCHITECTURE STYLE:
 1. openGo aims for maximum speed and simplicity of operation.
    For this reason you may encounter a global ctxBruh in redis,
    which actually cannot be touched, because it is just a stub
    so that the redis driver doesn't complain.
    Based on the same point, all of openGo is made as a monolith, no
    microservices or grpc here, because the main task is 100% emulation
    of the php backend.
 2. Weirdness is better than standards
    If it is convenient for me alone (or suddenly for another couple of contributors)
    to dig into a self-made architecture, rather than climb into
    go idioms or "corporate code", then I'd rather do it my way.
    All endpoints will lie in package main, and their file names will be "z*.go".
    This is done exclusively for convenience so as not to suffer with packages.
 3. Portability
    openGo must run everywhere - from windows or prod on freebsd,
    all the way to haiku Os or worse termux on a xiaomi for 5 thousand from a flea market.
    Portability pursues openGo even in architecture, in the future I
    plan to add a small layer of abstraction for hot-swapping
    net/http with some fasthttp or something in that spirit.
    Or even add an even thicker layer for hot-swapping
    nginx with go's built-in means of working with proxying and static files.
 4. Compatibility
    openGo must replace the php backend on the fly, without breaking anything in client logic.
    This means you will have to reverse the old php code (which thank god is not yet
    "scary legacy with schrodinger's classes") and repeat its behavior exactly
    in go handlers. This may be harder in the moment, but in the long term
    a conditional version 0.96.3 (from September 2025) will not even notice that it suddenly
    started working inside ojhub-loader-openGo and sends requests to the openGo backend
 5. Anti-testability
    You will ask "what are you talking about?", and I will answer "our test is whether this endpoint works in the client,
    and if it doesn't work then something is wrong". I just don't need to build all these
    mockings or whatever they write tests with. For me, just working capability
    on all current 5 versions of object hub is more than enough.
 6. Nastiness
    openGo will depend very heavily on redis in those places where it gives a win.
    I went further and instead of "unmarshalling => manipulations => marshalling" I
    will return bare bytes from redis in response, sometimes allowing myself
    to play with the actual string and jump by its indices.
    In theory this will give a 1-5% win, in practice I won't have to
    build unnecessary json.Marshal and just glue strings by hand,
    just like in the times of ojhub-php-build-120

TODO:
 1. fix lgbtBan values for countries where LGBT is prohibited
    (so that there are no projects about LGBT in search and generally so that Egyptians don't put me on a bottle)
 2. add redis when load appears
    (I was too lazy to add it because even 5 rps is an event for me)
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// last build number of client with php backend implementation - 133
// openGo will use the same routes as php
// below you will find the proxy mechanism that handles this
var (
	HELPER_VER   = "133"
	REVISION_NUM = "3"
	apiAddr      = "/server/" + HELPER_VER + "/"
	php          = ".php"
	HELPER_URL   = "https://objecthub.xyz/"
)

var LangList = []string{"RU", "EN", "UA"}

func IsValidLang(lang string) bool {
	for _, l := range LangList {
		if l == lang {
			return true
		}
	}
	return false
}

const (
	MAX_DESC_LEN  = 5000
	MAX_SHORT_LEN = 200
)

type Endpoint struct {
	// method must be specified with a space - "POST "
	// or if method is unimportant then literally empty string ""
	// again for convenience of route registration
	Method  string
	Path    string
	Handler func(http.ResponseWriter, *http.Request)
}

func DropPhp(w http.ResponseWriter, r *http.Request) {
	// while nginx is on production we give it a signal
	// "399 => okay I'm going to php instead of go"
	w.WriteHeader(399)
}

// Note - in php there are 97 endpoints, here there are only 92
var endpoints = []Endpoint{
	{"POST ", apiAddr + "user/login" + php, userLogin},                  // zUserLogin.go
	{"POST ", apiAddr + "user/logout" + php, userLogout},                // zUserLogin.go
	{"POST ", apiAddr + "user/register" + php, userRegister},            // zUserLogin.go
	{"POST ", apiAddr + "send/newsPost" + php, AddNews},                 // zNews.go
	{"POST ", apiAddr + "send/newsModify" + php, NewsEdit},              // zNews.go
	{"", apiAddr + "delete/newsPost" + php, NewsDelete},                 // zNews.go
	{"POST ", apiAddr + "send/comment" + php, CommentSend},              // zComms.go
	{"POST ", apiAddr + "send/commentModify" + php, CommentModify},      // zComms.go
	{"", apiAddr + "delete/comment" + php, CommentDelete},               // zComms.go
	{"POST ", apiAddr + "send/like" + php, LikeSend},                    // zLikes.go
	{"POST ", apiAddr + "send/dislike" + php, DislSend},                 // zLikes.go
	{"POST ", apiAddr + "reportGdps" + php, ReportGdps},                 // zReport.go
	{"", apiAddr + "user/devices" + php, OpenDeviceTab},                 // zDevices.go
	{"", apiAddr + "user/removeDevice" + php, RemoveDevice},             // zDevices.go
	{"POST ", apiAddr + "user/getAccInfo" + php, GetConfInfo},           // zUserEdit.go
	{"", apiAddr + "user/setNickname" + php, SetNicknameHandler},        // zUserEdit.go
	{"", apiAddr + "user/setSocials" + php, SetSocialsHandler},          // zUserEdit.go
	{"", apiAddr + "user/setResume" + php, SetResumeHandler},            // zUserEdit.go
	{"POST ", apiAddr + "send/deviceAdd" + php, DeviceAddTab},           // zDevices.go
	{"", apiAddr + "content/getAlarms" + php, GetAlarms},                // zAlarms.go
	{"", apiAddr + "content/getAlarm" + php, GetAlarm},                  // zAlarms.go
	{"", apiAddr + "send/deleteAlarm" + php, DeleteAlarm},               // zAlarms.go
	{"", apiAddr + "send/writeAlarm" + php, WriteAlarmHandler},          // zAlarms.go
	{"", apiAddr + "send/campAdd" + php, CampAdd},                       // zGdpsAdd.go
	{"", apiAddr + "send/showAdd" + php, ShowAdd},                       // zGdpsAdd.go
	{"", apiAddr + "send/pereAdd" + php, PereAdd},                       // zGdpsAdd.go
	{"", apiAddr + "send/teleAdd" + php, TeleAdd},                       // zGdpsAdd.go
	{"", apiAddr + "send/campEdit" + php, CampEdit},                     // zGdpsEdit.go
	{"", apiAddr + "send/showEdit" + php, ShowEdit},                     // zGdpsEdit.go
	{"", apiAddr + "send/pereEdit" + php, PereEdit},                     // zGdpsEdit.go
	{"", apiAddr + "send/teleEdit" + php, TeleEdit},                     // zGdpsEdit.go
	{"", apiAddr + "content/getJoinLog" + php, GetJoinLog},              // zOwners.go
	{"", apiAddr + "send/bump" + php, GdpsBump},                         // zGdps.go
	{"", apiAddr + "content/getOwners" + php, GetOwners},                // zOwners.go
	{"", apiAddr + "send/permAdd" + php, PermAdd},                       // zOwners.go
	{"", apiAddr + "send/perm" + php, PermRemove},                       // zOwners.go
	{"", apiAddr + "gdps/sub" + php, gdpsSub},                           // zAlarms.go
	{"", apiAddr + "gdps/unsub" + php, gdpsUnsub},                       // zAlarms.go
	{"", apiAddr + "gdps/subs" + php, gdpsSubs},                         // zAlarms.go
	{"POST ", apiAddr + "sub" + php, PushSub},                           // zPushs.go
	{"", apiAddr + "content/fetchComms" + php, LoadMoreComms},           // zComms.go
	{"", apiAddr + "content/newsAll" + php, GlobalNews},                 // zGlobalNews.go
	{"", apiAddr + "content/news" + php, LocalNews},                     // zGlobalNews.go
	{"", apiAddr + "content/newsC" + php, NewsGetOne},                   // zGlobalNews.go
	{"", apiAddr + "search/new" + php, fullSearch},                      // zSearch.go
	{"", apiAddr + "content/getUser" + php, GetUserHandler},             // zPubProf.go
	{"", apiAddr + "content/getAddedCamps" + php, GetAddedCampsHandler}, // zPubProf.go
	{"", apiAddr + "content/getAddedShows" + php, GetAddedShowsHandler}, // zPubProf.go
	{"", apiAddr + "content/getAddedPeres" + php, GetAddedPeresHandler}, // zPubProf.go
	{"", apiAddr + "content/getAddedTeles" + php, GetAddedTelesHandler}, // zPubProf.go
	{"", apiAddr + "content/getUserGuides" + php, GetUserGuidesHandler}, // zPubProf.go
	{"", apiAddr + "wiki/getWikis" + php, wikiSearch},                   // zSearch.go
	{"", apiAddr + "wiki/getWiki" + php, GuidesHandler},                 // zWiki.go
	{"", apiAddr + "wiki/getGuides" + php, GuidesHandler},               // zWiki.go
	{"", apiAddr + "wiki/getGuide" + php, GuideHandler},                 // zWiki.go
	{"", apiAddr + "send/newWiki" + php, HandleNewWiki},                 // zWikiDashboard.go
	{"", apiAddr + "send/editWiki" + php, HandleEditWiki},               // zWikiDashboard.go
	{"", apiAddr + "wiki/colors" + php, HandleWikiColors},               // zWikiDashboard.go
	{"", apiAddr + "wiki/setMainWiki" + php, HandleSetMainWiki},         // zWikiDashboard.go
	{"", apiAddr + "search/conntectWiki" + php, ConnectWiki},            // zGdpsWikiLinknig.go
	{"", apiAddr + "search/conntectContent" + php, ConnectContent},      // zGdpsWikiLinknig.go
	{"", apiAddr + "content/getGuidesAdmin" + php, GetGuidesAdmin},      // zWikiDashboard.go
	{"", apiAddr + "send/newGuide" + php, NewGuide},                     // zWikiDashboard.go
	{"", apiAddr + "send/editGuide" + php, EditGuideHandler},            // zWikiDashboard.go
	{"", apiAddr + "wiki/setWikiTag" + php, SetWikiTagHandler},          // zWikiDashboard.go
	{"", apiAddr + "wiki/templateGet" + php, TemplateGet},               // zWikiTemplates.go
	{"", apiAddr + "wiki/templatesGet" + php, TemplatesGet},             // zWikiTemplates.go
	{"", apiAddr + "wiki/templateSave" + php, TemplateSave},             // zWikiTemplates.go
	{"", apiAddr + "wiki/templateDelete" + php, TemplateDelete},         // zWikiTemplates.go
	{"", apiAddr + "forum/create" + php, ForumCreate},                   // zForums.go
	{"", apiAddr + "forum/createPost" + php, ForumCreatePost},           // zForums.go
	{"", apiAddr + "send/forumPost" + php, ForumCreatePost},             // zForums.go
	{"", apiAddr + "forum/getPost" + php, ForumGetPost},                 // zForums.go
	{"", apiAddr + "forum/getPosts" + php, ForumGetPosts},               // zForums.go
	{"", apiAddr + "vacans/getAll" + php, vacsSearch},                   // zSearch.go
	{"", apiAddr + "vacans/apply" + php, vacApply},                      // zVacApls.go
	{"", apiAddr + "vacans/removeApl" + php, vacUnapply},                // zVacApls.go
	{"", apiAddr + "content/vacsC" + php, GetOneVac},                    // zVacApls.go
	{"", apiAddr + "vacans/get" + php, VacsGet},                         // zVacans.go
	{"", apiAddr + "send/vacsAdd" + php, VacsAdd},                       // zVacans.go
	{"", apiAddr + "send/vacsEdit" + php, VacsEdit},                     // zVacans.go
	{"", apiAddr + "vacans/edit" + php, VacsEditPre},                    // zVacans.go
	{"", apiAddr + "vacans/applies" + php, VacsApplies},                 // zVacans.go
	{"", apiAddr + "vacans/removeVac" + php, VacsRemove},                // zVacans.go
	{"", apiAddr + "content/camp" + php, GDPSopener},                    // zGdps.go
	{"", apiAddr + "loginT" + php, FreeBSDcompile},                      // zLoginT.go
	{"", apiAddr + "likesT" + php, LikesT},                              // zLikesT.go
	{"", apiAddr + "wordleRU" + php, WordleRU},                          // zWordle.go
	{"", apiAddr + "wordleEN" + php, WordleEN},                          // zWordle.go
	{"", "/join" + php, GDPSjoin},                                       // zLoader.go
	{"", "/loader", CliLoader},                                          // zLoader.go
	{"", "/", IndexParser},                                              // zIndex.go
	{"", apiAddr + "challenge" + php, AltchaChallenge},                  // captcha.go
	{"", apiAddr + "!newTakeAll" + php, AdminTakeAll},                   // zAdmin.go
	{"", apiAddr + "Aaction" + php, Aaction},                            // zAdmin.go
	{"", apiAddr + "wiki/filesGet" + php, DropPhp},
	{"", apiAddr + "wiki/filesSend" + php, DropPhp},
	{"", apiAddr + "search/deleteWikiFiles" + php, DropPhp}, // why the hell is file deletion in search?
	{"", apiAddr + "vless" + php, DropPhp},
	{"", apiAddr + "user/works" + php, WorksGet},                    // zWorks.go
	{"", apiAddr + "user/workAdd" + php, workAddHandler},            // zWorks.go
	{"", apiAddr + "user/workEdit" + php, workEditHandler},          // zWorks.go
	{"POST ", apiAddr + "user/workDelete" + php, workDeleteHandler}, // zWorks.go
	{"POST ", apiAddr + "user/workVerify" + php, workVerifyHandler}, // zWorks.go

}

func main() {
	fmt.Printf("Object Hub openGo, Protocol %s Revision %s\n", HELPER_VER, REVISION_NUM)
	// for now I use redis. But in the future I may well
	// create my own cache right in the process,
	// because instance is 1, and I simply don't need to keep
	// a single service for in-ram cache
	InitDB()           // database.go
	InitGeoDb()        // databese.go
	rDb := InitRedis() // database.go
	InitAltcha()       // captcha.go
	defer DB.Close()
	defer GeoDb.Close()
	if rDb {
		defer RamDB.Close()
	}
	mux := http.NewServeMux()
	for _, ep := range endpoints {
		mux.HandleFunc(ep.Path, ep.Handler)
		//log.Println("==> Loading", ep.Path)
	}
	registerStaticRoutes(mux)     // standalone.go: /imgs/, /cli/, /sw.js from IMGS and VERS_DIR
	handler := blockDotfiles(mux) // standalone.go

	// nginx no longer stands before openGo — we hold :80 ourselves (only for ACME
	// HTTP-01 challenge) and :443 (TLS), certificate is issued and renewed
	// automatically via Let's Encrypt (see autocert.go).
	if os.Getenv("USE_TLS") == "1" {
		StartServerWithAutocert(
			handler,
			[]string{"objecthub.xyz", "www.objecthub.xyz"},
			"/var/db/ojhub-autocert",
		)
	} else {
		log.Println("> USE_TLS=0, starting plain HTTP on :80")
		log.Fatal(http.ListenAndServe(":80", handler))
	}
}

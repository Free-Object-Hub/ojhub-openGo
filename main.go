/*
Object Hub openGo
Contributors:
MIOBOMB - архитектура + начало работы + весёлая работа
DenisC - Весь алгоритм работы поиска
//
Author: MIOBOMB (2026)
ojhub-opengo - открытая реализация бекенда Object Hub на Go(lang)
Так как php/node бекенды этого сайта работают на freebsd мы намеренно
отказались от докеров и прочей инфраструктуры с целью тотальной портируемости
этой реализации бекенда, ибо наш freebsd сервер не умеет крутить Docker,
и многие "модные" технологии из мира линукса, такие как проклятый SystemD.
Ну и из личного, я - MIOBOMB, и у меня FreeBSD стоит на десктопе,
у вас вероятно стоит Windows, и у кого то ещё не настроен WSL2.
//
ЦЕЛЬ ПРОЕКТА НА БЛИЖАЙШИЕ 2 МЕСЯЦА:
Замена самых горячих участков на Golang, и избавление от nodejs части бекенда
с возвратом php на те места, которые go какое то время обслуживать не сможет
//
ДОЛГОСРОЧНАЯ ЦЕЛЬ ПРОЕКТА:
openGo целится на 100% совместимость и заменяемость php бекенда,
вплоть до того, что маршруты, входные и выходные данные не изменятся никак.
Учтите что для такой реализации эндпоинтов вам придётся разбираться в
исходном коде клиента! За основу openGo была взята версия 0.97.33 и 0.97.7,
потому так удастся воссоздать 100% эмуляцию php бекенда.
//
FIXME:
подумать над интеграцией nginx и redis в openGo бинарник
когда openGo будет близок к 100% готовности

СТИЛЬ КОДА И АРХИТЕКТУРЫ:
 1. openGo ставит своей целью максимальную скорость и простоту работы.
    По этой причине вы можете встретить глобальный ctxBruh в redis,
    который на самом деле нельзя трогать, потому что он просто затычка
    чтобы redis драйвер не ругался.
    Исходя из того же пункта весь openGo выполнен как монолит, никаких
    микросервисов или grpc тут нет, ибо главная задача - 100% эмуляция
    php бекенда.
 2. Странности лучше стандартов
    Если мне одному (или вдруг другой парочке контрибьюторов) будет
    удобно копаться в самопальной архитектуре, а не лезть в
    идиомы go или "корпоративный код", то я лучше сделаю так как мне надо.
    Все эндпоинты будут лежать в package main, а их название файлов будет "z*.go".
    Сделано это исключительно для удобства чтоб не мучиться с пакетами.
 3. Портируемость
    openGo должен запускаться везде - от windows или прода на freebsd,
    вплоть до haiku Os или того хуже termux на ксяоми за 5 тыщ с барахолки.
    Портируемость преследует openGo даже в архитектуре, в будущем я
    планирую добавить небольшой слой абстракции для горячей замены
    net/http на какой нибуть fasthttp или что то в таком духе.
    Или вовсе добавить ещё более толстый слой ради горячей замены
    nginx на встроенные в go средства работы с проксированием и статикой.
 4. Совместимость
    openGo должен заменять php бекенд на горячую, не ломая ничего в логике клиента.
    Это значит что вам придётся реверсить старый php код (который слава богу ещё не
    "страшное легаси с классами шрёдингера") и повторять его поведение точь в точь
    в go-хандлерах. Это в моменте может быть тяжелее, но в долгосрочной перспективе
    условная версия 0.96.3 (от сентября 2025) даже не заметит что она внезапно
    начала работать внутри ojhub-loader-openGo и отправляет запросы на openGo бекенд
 5. Анти тестируемость
    Вы спросите "ты чё несёшь?", а я отвечу "наш тест - работает ли этот эндпоинт в клиенте,
    и если не работает значит что то не так". Мне просто не нужно городить все эти
    моканья или как там пишут тесты. Мне за глаза просто работоспособности
    на всех текущих 5 версиях object hub.
 6. Мерзость
    openGo будет очень сильно зависеть от redis в тех местах где он даст выигрыш.
    Я же пошёл дальше и вместо "анмаршалинг => манипуляции => маршалинг" я
    буду отдавать голые байты из redis в ответ, иногда позволяя себе
    играться с настоящей строкой и прыгая по её индексам.
    В теории это даст выигрыш в 1-5%, на практике мне не придётся
    городить лишние json.Marshal и просто склеивать строки ручками,
    прям как во времеза ojhub-php-build-120

TODO:
 1. починить lgbtBan значения для стран где ЛГВТ запрещён
    (чтобы в поиске не было проектов про лгвт и вообще чтобы меня не посадили на бутылку)
 2. добавить redis когда появится нагрузка
    (я поленился его добавить потому что у меня даже 5 rps - событие)
*/
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	//
	_ "github.com/go-sql-driver/mysql"
)

// последний номер сборки клиента с php реализацией бекенда - 133
// openGo будет использовать те же маршруты что и php
// ниже вы найдёте прокси механизм который это и отрабатывает
var (
	HELPER_VER = "133"
	apiAddr    = "/server/" + HELPER_VER + "/"
	php        = ".php"
)

type Endpoint struct {
	// метод нужно указывать с пробелом - "POST "
	// либо если метод неважен то буквально пустая строка ""
	// опять же ради удобства регистрации маршрутов
	Method  string
	Path    string
	Handler func(http.ResponseWriter, *http.Request)
}

func DropPhp(w http.ResponseWriter, r *http.Request) {
	// пока на продакшне стоит nginx даём ему сигнал
	// "399 => окей иду в php вместо go"
	w.WriteHeader(399)
}

// Обратите внимание - в php 64 эндпоинта, здесь же их всего 30
var endpoints = []Endpoint{
	{"POST ", apiAddr + "user/login" + php, userLogin},             // zUserLogin.go
	{"POST ", apiAddr + "user/register" + php, userRegister},       // zUserLogin.go
	{"POST ", apiAddr + "send/newsPost" + php, AddNews},            // zNews.go
	{"POST ", apiAddr + "send/newsModify" + php, NewsEdit},         // zNews.go
	{"", apiAddr + "delete/newsPost" + php, NewsDelete},            // zNews.go
	{"POST ", apiAddr + "send/comment" + php, CommentSend},         // zComms.go
	{"POST ", apiAddr + "send/commentModify" + php, CommentModify}, // zComms.go
	{"", apiAddr + "delete/comment" + php, CommentDelete},          // zComms.go
	{"POST ", apiAddr + "send/like" + php, LikeSend},               // zLikes.go
	{"POST ", apiAddr + "send/dislike" + php, DislSend},            // zLikes.go
	{"POST ", apiAddr + "reportGdps" + php, ReportGdps},            // zReport.go
	{"", apiAddr + "user/devices" + php, OpenDeviceTab},            // zDevices.go
	{"", apiAddr + "user/removeDevice" + php, RemoveDevice},        // zDevices.go
	{"POST ", apiAddr + "user/getAccInfo" + php, GetConfInfo},      // zUserEdit.go
	{"POST ", apiAddr + "send/deviceAdd" + php, DeviceAddTab},      // zDevices.go
	{"", apiAddr + "content/fetchComms" + php, LoadMoreComms},      // zComms.go
	{"", apiAddr + "content/newsAll" + php, GlobalNews},            // zGlobalNews.go
	{"", apiAddr + "content/news" + php, LocalNews},                // zGlobalNews.go
	{"", apiAddr + "content/newsC" + php, NewsGetOne},              // zGlobalNews.go
	{"", apiAddr + "search/new" + php, fullSearch},                 // zSearch.go
	{"", apiAddr + "wiki/getWikis" + php, wikiSearch},              // zSearch.go
	{"", apiAddr + "wiki/getWiki" + php, GuidesHandler},            // zWiki.go
	{"", apiAddr + "wiki/getGuide" + php, GuideHandler},            // zWiki.go
	{"", apiAddr + "vacans/getAll" + php, vacsSearch},              // zSearch.go
	{"", apiAddr + "vacans/apply" + php, vacApply},                 // zVacApls.go
	{"", apiAddr + "vacans/removeApl" + php, vacUnapply},           // zVacApls.go
	{"", apiAddr + "content/vacsC" + php, GetOneVac},               // zVacApls.go
	{"", apiAddr + "content/camp" + php, GDPSopener},               // zGdps.go
	{"", apiAddr + "loginT" + php, FreeBSDcompile},                 // zLoginT.go
	{"", apiAddr + "likesT" + php, LikesT},                         // zLikesT.go
	{"", "/join" + php, GDPSjoin},                                  // zLoader.go
	{"", "/loader", CliLoader},                                     // zLoader.go
	{"", "/", IndexParser},                                         // zIndex.go

	// дальше идут эндпоинты которые на самом деле все ещё обслуживаются php
	{"", apiAddr + "content/getUser" + php, DropPhp},
	{"", apiAddr + "content/getAddedCamps" + php, DropPhp},
	{"", apiAddr + "content/getAddedShows" + php, DropPhp},
	{"", apiAddr + "content/getAddedPeres" + php, DropPhp},
	{"", apiAddr + "content/getUserGuides" + php, DropPhp},
	{"", apiAddr + "send/campAdd" + php, DropPhp},
	{"", apiAddr + "send/showAdd" + php, DropPhp},
	{"", apiAddr + "send/pereAdd" + php, DropPhp},
	{"", apiAddr + "send/teleAdd" + php, DropPhp},
	{"", apiAddr + "send/campEdit" + php, DropPhp},
	{"", apiAddr + "send/showEdit" + php, DropPhp},
	{"", apiAddr + "send/pereEdit" + php, DropPhp},
	{"", apiAddr + "send/teleEdit" + php, DropPhp},
	{"", apiAddr + "vacans/get" + php, DropPhp},
	{"", apiAddr + "vacans/edit" + php, DropPhp},
	{"", apiAddr + "send/vacsAdd" + php, DropPhp},
	{"", apiAddr + "vacans/removeVac" + php, DropPhp},
	{"", apiAddr + "vacans/applies" + php, DropPhp},
	{"", apiAddr + "content/getJoinLog" + php, DropPhp},
	{"", apiAddr + "search/connectWiki" + php, DropPhp},
	{"", apiAddr + "send/newWiki" + php, DropPhp},
	{"", apiAddr + "send/editWiki" + php, DropPhp},
	{"", apiAddr + "wiki/colors" + php, DropPhp},
	{"", apiAddr + "content/getGuidesAdmin" + php, DropPhp},
	{"", apiAddr + "send/newGuide" + php, DropPhp},
	{"", apiAddr + "send/editGuide" + php, DropPhp},
	{"", apiAddr + "wiki/setWikiTag" + php, DropPhp},
	{"", apiAddr + "wiki/templatesGet" + php, DropPhp},
	{"", apiAddr + "wiki/templateSave" + php, DropPhp},
	{"", apiAddr + "wiki/filesGet" + php, DropPhp},
	{"", apiAddr + "wiki/filesSend" + php, DropPhp},
	{"", apiAddr + "content/getOwners" + php, DropPhp},
	{"", apiAddr + "send/perm" + php, DropPhp},
	{"", apiAddr + "send/permAdd" + php, DropPhp},
	// Админ панель
	{"", apiAddr + "!newTakeAll" + php, DropPhp},
	//{"", apiAddr + "" + php, DropPhp},
}

func main() {
	// пока что я использую redis. Но в будущем я вполне могу
	// и создать свой кеш прямо в процессе,
	// потому что экземпляр 1, и держать единый сервис для in-ram cache
	// мне просто не надо
	InitDB()    // database.go
	InitGeoDb() // databese.go
	InitRedis() // database.go
	defer DB.Close()
	defer GeoDb.Close()
	defer RamDB.Close()

	for _, ep := range endpoints {
		http.HandleFunc(ep.Path, ep.Handler)
		log.Println("==> Loading", ep.Path)
	}
	server := &http.Server{
		Addr: ":8080",
	}
	// Запускаем сервер отдельно,
	// чтобы main мог дождаться сигнала завершения.
	go func() {
		log.Println("> Starting on :8080")
		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	// Ждём Ctrl+C / SIGTERM
	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		os.Interrupt,
		syscall.SIGTERM,
	)
	<-signalChan
	log.Println("> Shutting down...")
	// Даём текущим HTTP-запросам закончиться.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Println("Shutdown error:", err)
	}
	log.Println("> Shutdown complete")
}

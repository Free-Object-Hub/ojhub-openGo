package main

import (
	"net/http"
)

func PushSub(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	user, ok := RequireDeviceForInactiveAccs(w, r)
	if !ok {
		return
	}
	endpoint := r.FormValue("endpoint")
	p256dh := r.FormValue("p256dh")
	auth := r.FormValue("auth")

	if endpoint == "" || p256dh == "" || auth == "" {
		w.Write([]byte("-3"))
		return
	}
	if len(endpoint) > 500 {
		w.Write([]byte("-2"))
		return
	}

	deviceToken := GetDeviceToken(r) // вообще это мидлвар но кто запрещает пинать напрямую
	device, err := GetDeviceByToken(deviceToken)
	if err != nil {
		w.Write([]byte("-5"))
		return
	}

	userAgent := r.Header.Get("User-Agent")
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}

	if err := SaveSubscription(user.UserId, device.ID, endpoint, p256dh, auth, userAgent); err != nil {
		w.Write([]byte("-1"))
		return
	}

	w.Write([]byte("1"))
}

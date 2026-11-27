package main

import (
	"log"
	"strconv"
	"sync"
	"time"
)

type ramItem struct {
	val string
	exp int64 // unix nano, 0 = без срока (как ttl=0 в redis)
}

const ramItemOverhead = 64 // грубая оценка накладных расходов map + структуры

var (
	ramMu    sync.RWMutex
	ramMap   = make(map[string]ramItem)
	ramBytes int64
	ramMax   int64 = 64 << 20 // 64 МБ по умолчанию
	ramOn    bool
	ramOnce  sync.Once
)

func itemSize(key, val string) int64 {
	return int64(len(key) + len(val) + ramItemOverhead)
}

func RamGet(key string) (string, error) {
	if !ramOn {
		return "", nil
	}
	ramMu.RLock()
	it, ok := ramMap[key]
	ramMu.RUnlock()
	if !ok {
		return "", nil
	}
	if it.exp != 0 && time.Now().UnixNano() > it.exp {
		return "", nil // протухло; удалит janitor или перезапись
	}
	return it.val, nil
}

func RamSet(key string, val string, ttl time.Duration) error {
	if !ramOn {
		log.Println("ram cache is off")
		return nil
	}
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	ramMu.Lock()
	if old, ok := ramMap[key]; ok {
		ramBytes -= itemSize(key, old.val)
	}
	ramMap[key] = ramItem{val: val, exp: exp}
	ramBytes += itemSize(key, val)
	if ramBytes > ramMax {
		ramEvictLocked()
	}
	ramMu.Unlock()
	return nil
}

func RamDel(keys ...string) error {
	if !ramOn {
		return nil
	}
	ramMu.Lock()
	for _, k := range keys {
		if old, ok := ramMap[k]; ok {
			ramBytes -= itemSize(k, old.val)
			delete(ramMap, k)
		}
	}
	ramMu.Unlock()
	return nil
}

// вызывать под ramMu.Lock()
func ramEvictLocked() {
	now := time.Now().UnixNano()
	// 1) сначала выкидываем протухшее
	for k, it := range ramMap {
		if it.exp != 0 && now > it.exp {
			ramBytes -= itemSize(k, it.val)
			delete(ramMap, k)
		}
	}
	// 2) если всё ещё много — режем случайные до 90% лимита
	target := ramMax / 10 * 9
	for k, it := range ramMap {
		if ramBytes <= target {
			break
		}
		ramBytes -= itemSize(k, it.val)
		delete(ramMap, k)
	}
}

func ramJanitor() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now().UnixNano()
		ramMu.Lock()
		for k, it := range ramMap {
			if it.exp != 0 && now > it.exp {
				ramBytes -= itemSize(k, it.val)
				delete(ramMap, k)
			}
		}
		ramMu.Unlock()
	}
}

func InitRamCache() bool {
	if mb, err := strconv.Atoi(getEnv("RAMDB_MAX_MB", "64")); err == nil && mb > 0 {
		ramMax = int64(mb) << 20
	}
	ramOn = true
	ramOnce.Do(func() { go ramJanitor() })
	log.Printf("> RAM cache done (limit %d MB)", ramMax>>20)
	return true
}

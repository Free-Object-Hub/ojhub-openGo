# ojhub-openGo

ojhub-opengo - open implementation of Object Hub backend in Go, which has already replaced the nodejs backend and runs in production.

# Who are we anyway?
Object hub - a platform for the Object Show Community, with emphasis on its own wiki engine and media. We deliberately abandoned popular web standards in favor of site performance - in theory the Object hub client interface runs at a stable 60 fps even on weak Chromebooks (because the entire engine was written on an Athlon 64 X2)!

# Why openGo
nodejs 150 megabytes RAM idle, openGo 13 megabytes idle, as they say - think about it.  
Correction from mid-September - openGo became Standalone, but its idle is still 25 megabytes

# A bit of reference information
openGo was made in 8 days based on a miraculously found prototype on an almost dead HDD.  
In these 8 days I vibe-coded 50% of Object hub API and rolled it out to production.  
Since the php/node backends of this site run on FreeBSD we deliberately abandoned excessive infrastructure for the purpose of total portability of this backend implementation.

## Why Russian language?
The Object hub engine was historically created back when I (MIOBOMB) was 15 years old and I barely understood English, the reasons why comments in code and documentation are still written in Russian:
- Historical layer (everything is already in Russian everywhere)  
- I'm still not strong in English  
- Maintaining a relatively unified style everywhere  

## Current infrastructure:
- Go 1.26
- Mariadb 11.8, but any other modern version can be used
- (optional) redis 8.8.0

## Long-term goal (almost closed):
- Minimal resource consumption
- High portability
- 100% compatibility with legacy-php
- 100% compatibility with supported versions of object hub
- Normal and documented infrastructure
- MaxMind GeoLite2

## checklist (chatgpt suggested I add)
- Production? yes
- Loader 1.10? Already 1.40
- Useful links? well, sorry I'm still not good at organizing repositories
- How much API is ready? 92 / 97 endpoints
- What will be ported next? wiki files
- What will be fixed next? I'll add redis where I was lazy because "eh the load is nothing anyway"
- What will I do when I'm in the mood? install.sh script, with support for freebsd, debian linux, termux

# Object Hub Protocol V133 R3
  
ATTENTION!!! THIS DOCUMENT IS UNFINISHED, DON'T BE SURPRISED BY BROKEN LINKS  
Version 133 of the Object Hub protocol, new revisions are backward compatible for target clients  
The first client version on this protocol is 0.97.4  
This will not be a description of how the backend works, but a description of the PROTOCOL, just like the X11 protocol, which doesn't know what a video card is  
  
This may also contain a detailed description of the divergences between the protocol and the Object Hub openGo project  
  
# Table of contents  

- Revisions
- - R0
- - R1
- - R2
- - R3
- Philosophy
- History
- - Historically-grown-problems
- GDPS-Helper-Engine
- Functional-domains
- Meta-domains
- - Authorization
- - Init-endpoint
- - Likes
- - Comments
  

# Revisions
  
A revision is a backward-compatible extension of the protocol  
Revisions are denoted as R<N>  
  
Adding new functionality to the backend inevitably leads to an increase of the revision number  
If backward compatibility is broken in any part of the protocol when creating a new revision, it is a new version, not a revision  
  
## R0
The canonical revision of version 133, released together with Object Hub 0.97.4 NODE\_BETA1  
  
## R1
Released with Object Hub 0.97.4 NODE\_BETA3  
Extensions:  
- Support for sending user&device tokens via the body, alongside headers. Added to preserve compatibility with protocol version 132  
- likesT.php endpoint - a hashset returning absolutely all records of the user's like statuses, used to reconcile the like state on the client (version 133 originally wasn't supposed to reconcile likes on the server, but I forgot to add a client-side cache and had to extend the revision)  
  
## R2
Released with Object Hub 0.98  
Extensions:  
- Project subscriptions  
- Many endpoints got error codes, they still don't have a clear record in the protocol  
  
## R3
Will be completed with the release of Object Hub 0.99, its temporary version is already available since Object Hub 0.98.2  
Extensions:  
- The entire portfolio infrastructure  
Changes:  
- The resume field in the profile will be empty by default, since it is no longer rendered as of 0.99; instead it will be in the portfolio by default  
- - Note that the field will be empty, not removed. This is done to formally preserve compatibility with old clients

# Philosophy
Throughout the history of Object Hub/GDPS Helper, the client and the server developed in parallel and in close coordination, with the goal of minimizing traffic, the number of API requests, the number of bytes in a response, and, in the end, your server  
For this reason the whole protocol will look very ugly and strange: it was built strictly to the requirements of the official Object Hub client, not to be a REST API  
And the very purpose of the protocol's existence is to optimize your backend and the network consumption of the Object Hub API (yes, I've said this out loud before, repetition is the mother of learning)
  

# History
Object Hub is a fork of GDPS Helper, and it had its own problems and quirks in the protocol  
The very first version of the Object Hub protocol was GDPS Helper 1.7, which implemented protocol V64 (very unstable at the time)  
By GDPS Helper 1.8 the protocol had partially settled - the domains were removed for good, and the first version of the Legacy PHP backend was ready  
After forking the beta version of GDPS Helper 1.8, in Object Hub I started breaking the protocol "for beauty" and "for convenience", breaking some of the aggressive optimizations (not absolutely everything, as it may seem at first)  
The protocol relatively settled only by version 104; all Object Hub clients from build 104 (client version 0.94) will most likely work without major complaints, or with minimal monkey-patches  
The fork of GDPS Helper Legacy PHP, which survived many Object Hub-specific updates, became the basis of Object Hub Protocol V133 R0  
  
## Historically-grown-problems
- The protocol is not unified enough
- - Some entities respond with objects, others with arrays
- Error formats differ from endpoint to endpoint, so each endpoint will have its own error table
- - Although in many places the response is fairly unified - 200, `Access denied`
- The problems listed above will not be fixed, in order to preserve compatibility with outdated clients  
- Since V133R1 the protocol has taken on preserving compatibility with much older versions, because of this a huge number of endpoints have a "free method": all Object Hub clients 0.97.33 and older either didn't send the token and the method was GET, or sent the token in the body and the method became POST
  
# GDPS-Helper-Engine
Many of the strange, stupid and non-obvious things come specifically from the GDPS Helper Engine (GHE). Forgive the tautology, but our homegrown engine lets us create not only powerful and clever optimizations, but also genuinely strange things. This section mostly lists the powerful features that explain some of the cringe parts of the protocol description
  
## newHelper lang
It is a perfectly ordinary l10n implementation written from scratch. Its philosophy is "let's make the client reactive and able to work without a server"  
  
The protocol knows very few details about newHelper lang, only because I use it for optimization tricks  
  
### Tags
Tags in projects (as on the whole site) are perhaps the most absurd part, because they are structured in a very complicated way  
  
Tags depend entirely on newHelper lang. The server stores and returns only tag IDs, while the tag texts live on the client in the translation dictionary. The tag ID is substituted into the translation variable using this formula:
```
(channel_name) + tag + (tag_id)
```
This is how `Showtag3` = "Series"  
  
Channel name in the formula:  
TODO: table "channel letter - channel number - prefix in the formula" for `c`, `s`, `p`, `t`  
Channel `p` (dubbing/translation) reuses the tags of channel `s`, i.e. the `Show` prefix is used for it  
  
MUST: return tags as JSON-inside-string with IDs, for example `"[\"1\",\"3\",\"9\"]"`  
MUST: not return tag texts, only the client knows them  
  
#### Regular tags
The `tags` field in the project entity. These are mostly genres and subgenres specific to the OSC (Object Show Community)  
  
#### Alternative tags
The `os` field in the project entity. It is a leftover from GDPS Helper that still makes sense. On GDPS Helper regular tags meant the GDPS server type, and alternative tags meant the supported platforms  
For example, the current Object Hub:  
- Channel `s` (show) - episode length  
  
## newHelper langs dictionary
A full-fledged multilingual description of an entity. Thanks to it, the client can keep the translation consistent if the project author translated the description into many languages  
  
Example of an old text field and a newHelper langs dictionary:  
```
{
  "title": "Привет мир!",
  "text": "{\"RU\": \"Привет ГХЭ!\",\"EN\": \"Hello GHE!\",\"ES\": \"¡Hola GHM!\"}"
}
```
newHelper langs can work with "legacy fields" - places where the text has not yet been moved to the new storage system and is returned as plain text  
  
Note that the new field arrives as a string with JSON content. The client tries to parse the string as JSON, and if it gets an error, it shows the field content directly; this behavior is exactly what creates the legacy fallback  
  
The format is detected by the first character of the value:  
```
if (text[0] === '{') {...}
```
MUST (server): a string with a dictionary starts with `{`, with no whitespace or BOM before it, otherwise the client will treat it as legacy text  
NOTE: legacy text that is itself a valid JSON object will be taken for a dictionary. In practice this doesn't occur  
  
### Object form
Starting with revision TODO the client understands both forms: JSON inside a string and a plain JSON object. This is an extension, not a compatibility break, because the forms are interchangeable and a backend implementation may choose whichever is more convenient  
SHOULD: return only the string to clients older than this revision  
  
### Where it is used
The list of dictionary fields and the supported languages are described in the newHelper lang section (not to be confused with newHelper langs)  


# Functional-domains  
## Meta-domains
- Authorization
- Init-endpoint
- Likes
- Comments
- Search
## Users
- Portfolio
- Devices
- Notifications + Pushes
## Projects
- News
- Vacancies
- Subscriptions (directly trigger Push notifications)
- Co-owners
## Wiki
- Forums
- Co-owners
- Article tags (need to be renamed to addresses)
- Articles
- Comic reader
- Templates
  
## Explaining the cringe:  
- The co-owner systems of GDPSes and Wikis will differ in the future, so it is not a meta-domain
- The uploader is just an uploader, it shouldn't be a domain because it is not "business logic", it is also not tied to a specific revision and may change internally however I need, although I don't see much point in that right now

# Meta-domains
  
These are entities that permeate all domains and other entities of the Object Hub protocol. It is still the most controversial functional domain  
  
Likes, comments, and search are domains that implement their internals based on a field that defines the type (likes put a like, search looks for projects rather than everything, a comment is placed under a news item rather than who knows where)  

## Authorization
It's not really a domain, but it is a very important part of the functionality of half of the API endpoints  
Authorization checks the user's permissions on a given entity and other execution conditions, and if the check fails it returns an error (usually the response "Access denied" without additional headers)  
Authorization is divided into 4 methods:
0. Public data
  There are no permission checks here, this data is available even to a scanner bot that for some reason decided to scan port 443 at the address `/server/133/search/new.php` of your server  
1. User data
  This authorization method usually only checks the validity of the user's token, found on Read endpoints with private data  
  For example - the device list, the local likes database  
2. Updating private data
  This authorization method checks the user token and the device token, ignoring the account activation status, found on all private Write endpoints  
  For example - deleting a device, changing the account password  
3. Updating public data
  This authorization method checks the user token and the device token, verifying that the account activation status is not zero, found on all public Write endpoints  
  For example - setting a like, posting a comment, adding a project  
The existence of method 1 is justified by database load optimization
  
### Ways to send tokens
MUST: the permission check treats tokens sent by different methods identically  
SHOULD: tokens from headers are checked first  
  
1. headers  
  The user token goes in the `user-token` header, the device token in the `device-static` header  
2. body  
  The user token goes in the `token` field of the request body, the device token goes in the `device` field of the request body  

## Init-endpoint
The endpoint responsible for initializing the official Object Hub client, usually contains a cache of private data and a bit of public data  
Historically it is the response not only for `loginT.php`, but for many other endpoints too, and sometimes with strange semantics  
  
Authorization method - 2. This particular method is used here to block the account on the client under normal circumstances, see error types  
Method - free  
No parameters  
  
MUST:  
- Skip the device check and return the user token in the token field inside Current\_user when the Init function is used on registration and login  
- Not touch the public part of the response regardless of any authorization errors  
SHOULD:  
- When using a cache, split public and private data into 2 separate caches, so as not to give one user another user's data  
- When the guest user is used in the response, don't cache it but store it directly in your backend's cache  
  
Response format:
```
[
  // start of private cache
  Current_user,
  [
    Projects_owned_by_user{},
    Wikis_owned_by_user{}
  ],
  // end of private cache, start of public cache
  Home_page_projects_cache{},
  Home_page_news_cache[],
  // end of public cache
]
```
Note that where it says object, the response comes as an object with keys:  
```
[
  // private part of the response
  User,
  [
    {
      "c1":Project,
      "p3":Project,
      "t4":Project
    },
    {
      "w1":Wiki,
      "w2":Wiki,
      "w3":Wiki
    }
  ],
  // public part of the response
  {
    "c1":Project,
    "s2":Project
  },
  [
    News,
    News
  ]
]
```
To understand what the letters in the response mean, see the "Projects" section  
  
Error format - usually, if an error occurred on the server side while authorizing the user (for example an expired device token), a "guest user" is returned to the user, i.e. a regular stub is sent instead of the current user, where the token field literally contains the string "false" (not bool!), and the array of the user's projects and wikis will be empty - [{},{}]  
If there are no tokens at all (a user without an account), the guest user won't have a token field in the response at all  
Types of degradation:  
1. Expired tokens - 200, guest user + token:"false"  
2. Tokens missing - 200, guest user + token absent  
3. Failed to determine city and country by IP address - the cityData field degrades, the field becomes ["Unknown","Unknown"]  

## Likes
The domain responsible for checking likes on a specific element of an entity  
MUST: not allow likes to be sent twice  
SHOULD NOT: count the number of likes by scanning the status table on every action (even a simple project open)  
  
## Object Hub like types
- This is a separate Enum-like space  
0 = Projects  
1 = Project comments  
2 = News  
3 = Dead field, formerly GDPS Helper texture packs until their removal in GDPS Helper 1.9  
4 = Dead field, formerly comments on GDPS Helper texture packs until their removal in GDPS Helper 1.9  
5 = News comments  
6 = Wiki article comments  
7 = Wiki articles  
8 = Wikis  
9 = Forum posts  
10 = Forum post comments  
11 = Vacancies  
12 = Vacancy comments  
  
## Like rollback
Each entity object stores its own list of likes and dislikes separately, the like status is stored in a hashmap/table with indexes  
If there is a record of an existing like and an attempt is made to set a new rating, the rating counts will change in the reverse order based on the already existing record in the hashmap, then the record from the like status table will be deleted  
  
## Endpoints
  
### /server/133/send/like.php
Increments the like counter  
  
Authorization method - 3  
Method - free  
Parameters (body or query, doesn't matter):  
- ide = Entity ID  
- type = Entity type (in Object Hub terms type=0 is a project, and type=1 is project comments)  
  
Response format:  
```
[
  Like count,
  Dislike count
]
```  
openGo error types, not part of the canonical protocol:  
1. Any of the tokens is invalid - 200, `Access denied`  
2. Invalid ide - 500, `invalid id`  
3. Invalid type - 500, `invalid type`  
4. type is not in the table of registered channels - 500, `Unknown like type`  
  
### /server/133/send/dislike.php
Increments the dislike counter  
  
Authorization method - 3  
Method - free  
Parameters (body or query, doesn't matter):  
- ide = Entity ID  
- type = Entity type (in Object Hub terms type=0 is a project, and type=1 is project comments)  
  
Response format:  
```
[
  Like count,
  Dislike count
]
```  
openGo error types, not part of the canonical protocol:  
1. Any of the tokens is invalid - 200, `Access denied`  
2. Invalid ide - 500, `invalid id`  
3. Invalid type - 500, `invalid type`  
4. type is not in the table of registered channels - 500, `Unknown like type`  
  
### /server/133/likesT.php
A hashset cache for coloring the like buttons in the Object Hub client  
It became a separate endpoint from loginT.php in order to balance on TCP handshakes  
  
Authorization method - 1, in practice openGo implements 2  
Method - free  
No parameters  
  
Response format:  
```
{
  "p": projects,
  "c": comments,
  "n": news,
  "g": "guides" - wiki articles,
  "w": wikis,
  "f": forum posts,
  "v": vacancies,
  "subs": project subscriptions, which exploit the likes infrastructure purely because of the similar storage mechanics, appeared in Revision 2
}
```
Error types:  
1. Any of the tokens is invalid - 200, `Access denied`  
2. The account does not exist - 200, `Access denied`  
  

## Comments
A cross-cutting domain responsible for storing and handling comments
  
### The "Comment" entity  
```
[
  ID,
  Author_nick,
  Text,
  Author_UserID,
  Author_role,
  [
    Like_count,
    Dislike_count
  ],
  Date_in_unix,
  0 (dead field, formerly a flag "the user has a like here", this behavior remained in Legacy PHP)
]
```  
  
### "Standard comment getter"
This is an internal method (or function) of your backend, which must strictly follow this positional argument format:  
`Function(Channel, ID, Page)` (in Legacy PHP the third argument is userId, and only the fourth is Page)  
  
This function must return a list of 11 comments if possible. The offset works like this: ` OFFSET = page * 10`  
This is because the client checks for the presence of an 11th element and, if it exists, resets the render and inserts a "show more" button; effectively this means that to save traffic you may not return a full 11th comment, and just put a 0 there (if you chose to respond with an array)  
Response formats that are acceptable for the client:  
```
[
  Comment,
  Comment,
  ...
]
```
```
{
  "c<ID>": Comment,
  "c<ID>": Comment,
  ...
}
```
For an empty result `[]` or `{}` is returned, it makes no difference to the client  
  
### Object Hub comment channels
- This is a separate Enum-like space  
0 = Projects, likes channel = 1  
1 = Dead field, formerly GDPS Helper texture packs until their removal in GDPS Helper 1.9, likes channel = 4  
2 = "guides" - wiki articles, likes channel = 6  
3 = news, likes channel = 5  
4 = forum posts, likes channel = 10  
5 = vacancies, likes channel = 12  
  
### /server/133/send/comment.php  
Posts a comment under an entity and returns the current list of comments of the first page of this entity  
  
Authorization method - 3  
Method - free  
Parameters (body or query, doesn't matter):  
- ide = ID of the entity under which the comment is placed  
- type = Comment channel  
- text = Comment text  
  
MUST:  
- Sanitize text on the server side before checking the length  
- Not create a comment if the sanitized text is shorter than 10 BYTES (UTF-8), not characters. Cyrillic takes 2 bytes per character  
- Coerce an invalid type (not a number) to channel 0  
- Coerce type=1 (dead channel) to channel 0  
- Return the list of comments with status 200 even if the comment was not created due to rate limiting. The client doesn't learn that the comment was dropped  
SHOULD:  
- Rate-limit comment posting per user. openGo: 1 comment per 5 minutes, Legacy PHP: 1 comment per 10 minutes 
- Not depend on a specific list of channels: an unknown channel is not an error, the comment goes to channel 0, so that a new channel can't be "forgotten to register" and produce an error  
  
Response format:  
Standard comment getter  
  
Error types:  
1. Any of the tokens is invalid or the account is not activated - 200, `Access denied`  
2. Invalid ide (not a number or 0) - 400, `Invalid id` (openGo only)  
3. Text shorter than 10 bytes after sanitizing - 200, `-4`  
4. Internal error writing or reading comments - 500, error text (openGo only)  
  
### /server/133/content/fetchComms.php  
Returns a page of comments under an entity. Used by the client to load subsequent pages  
  
Authorization method - 0  
Method - GET  
Parameters (query):  
- type = Comment channel  
- id = ID of the entity under which the comments lie  
- page = Page number, counting from 0  
  
MUST:  
- Coerce invalid (non-numeric or missing) type, id and page to 0  
  
Response format:  
Standard comment getter  
  
openGo error types, not part of the canonical protocol:  
1. Internal error reading comments - 500, `Failed to get comments: <error text>`  
2. Internal error serializing the response - 500, `Failed to encode comments: <error text>`  
  
### /server/133/send/commentModify.php  
Changes the text of an existing comment and returns the new text  
  
Authorization method - 3  
Method - free  
Parameters (body or query, doesn't matter):  
- id = ID of the comment itself (not of the entity under which it lies)  
- text = New comment text  
  
MUST:  
- Allow editing a comment only to its author or a moderator (a user with non-zero priority). A moderator can edit any comments  
- Sanitize text on the server side before saving  
- NOT update the comment's Date\_in\_unix on edit  
- Respond with the raw text from the request with status 200 on success. The client shows this response instead of the old text  
  
Response format:  
```
New comment text, plain string
```
Error types:  
1. Any of the tokens is invalid - 200, `Access denied`  
2. Invalid id (not a number) - 400, `Invalid id` (openGo only)  
3. Account not activated - 200, empty response  
4. Text shorter than 10 bytes before sanitizing - 200, `-4`  
5. The comment belongs to another user (and the user is not a moderator) or does not exist - 200, `-3`  
6. Internal error reading or writing the comment - 500, error text (openGo only)  
  
### /server/133/delete/comment.php  
Deletes a comment along with its likes and returns the ID of the deleted comment  
  
Authorization method - 3  
Method - free for tokens, but ide MUST be passed in the query  
Parameters:  
- ide (query) = ID of the comment itself (not of the entity under which it lies)  
- type (query) = Comment channel. The client sends it, but openGo ignores it, the channel is taken from the comment itself  
  
MUST:  
- Allow deleting a comment only to its author or a moderator (a user with non-zero priority). A moderator can delete any comments  
- Delete the like/dislike records for this comment together with the comment
SHOULD:  
- Determine the channel from the comment itself, not from the type parameter: the client may send a wrong type and delete likes from the wrong space  
  
Response format:  
```
ID of the deleted comment, plain string
```
  
Error types:  
1. Any of the tokens is invalid - 200, `Access denied`  
2. The comment belongs to another user (and the user is not a moderator) - 200, `-1`  
3. Invalid ide (not a number) - 400, `Invalid id` (openGo only)  
4. Internal error reading or deleting the comment - 500, error text (openGo only)  
  

# Projects
The main entity of Object Hub, around which everything revolves  
  
## Project channels
"c" - camp (channel 0)  
"s" - show (channel 1)  
"p" - dubbing/translation (in the client code it is more often called Pere, less often Dub) (channel 2*)  
"t" - telegram channel (channel 3)  
  
\* - reuses the tags of channel 1  
  
## Tags
The `tags` and `os` fields contain arrays of tag IDs as strings. Details and the formula for getting the tag text are described in the newHelper lang section  
  
## Related entities
News - entirely tied to the gdpsId field  
Vacancies - entirely tied to the gdpsId field  
Wiki - can be "linked" to any project  
  
## Short entity "project"
Used when rendering in search  
```
"(channel letter) + (project id)":{
  ID - gdpsId of the project
  title - project title
  text - short project description, is a newHelper langs dictionary of added languages
  tags - project tags:
  [
    "1",
    "3",
    "9"
  ]
  os - alternative project tags:
  [
    "10",
    "11"
  ]
  likes: [like_count, dislike_count, comment_count]
  author - userId of the project author
  username - nickname of the project author
  img - project avatar
  ban - project banner
  channel - number of the project channel
  wiki - ID of the wiki linked to this project
}
```
  
## Full entity "project"
Used when opening the project page
```
"gdps": {
  ID - gdpsId of the project
  title - project title
  text - full project description, is a newHelper langs dictionary of added languages
  tags - project tags
  os - alternative project tags
  links - external links to the project itself:
  {
    "discord": "url",
    "youtube": "url"
  }
  likes: [like_count, dislike_count, comment_count]
  author - userId of the project author
  username - nickname of the project author
  img - url of the project avatar
  ban - url of the project banner
  freejoin - whether to show the join button or not (int)
  language - semi-dead field, formerly denoted the project language, since the second revision "MULTI" by default
  channel - number of the project channel
  wiki - ID of the wiki linked to this project
}
```

### /server/133/content/camp.php
Returns everything needed to render a project page: the full project entity, the first page of comments, and the project's news  
Despite the name, it serves projects of all channels (c, s, p, t), because gdpsId is unique across all channels  
  
Authorization method - 0  
Method - free  
Parameters:  
- id (query) = gdpsId of the project  
  
MUST:  
- Hide the project with the response `["NONE"]` if the user has the LGBT content ban enabled and the project has hasLgbt set. The response is indistinguishable from "project not found"  
- Return comments in the "Standard comment getter" format (channel 0, first page)  
SHOULD:  
- Not make the response depend on the user, so that it can be cached as a whole. openGo does not personalize the response and caches the project entity and the news (10 minutes each)  
  
Response format:  
```
{
  "gdps": Full_entity_project,
  "comments": Standard_comment_getter,
  "news": News
}
```
News are allowed in two forms, the client understands both:  
```
{
  "n<ID>": News_item,
  "n<ID>": News_item
}
```
```
[
  News_item,
  News_item
]
```
PHP returns news as an object, openGo returns a plain array as an optimization  
Comments in PHP also arrive as an object (`"c<ID>"`), in openGo the form is determined by the getter  
  
Error types:  
1. Project not found - 200, `["NONE"]`  
2. Project hidden due to the LGBT content ban - 200, `["NONE"]` (not implemented in openGo)  
3. Invalid id (not a number) - openGo: 400, `Invalid id parameter`, PHP: TODO verify (most likely `["NONE"]`)  
4. Any internal error (DB, Redis, serialization) - openGo: 200, `["NONE"]`. The client can't tell it apart from "project not found"  

### /server/133/send/bump.php
Bumps a project in the listing. A project can be bumped once every 2 hours, by any user  
  
Authorization method - 3  
Method - free  
Parameters:  
- id (query) = gdpsId of the project  
- tokens: sent as described in the Authorization section  
  
MUST:  
- Only bump projects with checked = 1  
- Treat the 2-hour window as per project, not per user  
- If the bump is still active (points > current time), not change points and return the current values  
- Otherwise set points to the current time + 7200  
  
Response format:  
```
[
  points,
  new_points_value,
  current_time - points
]
```
Where:  
- points = unix time until which the project is considered bumped, after this request  
- new_points_value = current time + 7200, calculated regardless of whether the bump happened  
- the third element is never positive: it is -7200 right after a successful bump, otherwise it is the negated number of seconds until the project can be bumped again  
  
A fresh bump can be told from an active one by `points == new_points_value`  
  
Error types:  
1. Any of the tokens is invalid or the account is not activated - 200, `Access denied`  
2. id is not a number, or the project has not passed the check (checked != 1) - 200, `no`  
3. The project does not exist or an internal error occurred - 200, empty response  
  
NOTE: project ownership is not checked, any activated user can bump any checked project  
NOTE: the response ends with a newline  
  
TODO: confirm that bumping by non-owners is intended  
TODO: check against the client how it tells "bumped" from "bump still active"  
TODO: where points go (search sorting?)  
  
### /join.php
Redirects to the project's external link and writes the join to the join log. Lives outside /server/133/. Unlike the rest of the protocol it answers with HTML or a redirect, not JSON or plain strings  
  
Authorization method - non-standard, depends on the freejoin field of the project  
Method - free  
Parameters:  
- id (query) = gdpsId of the project  
- m (query) = 1 if the join comes from the catalog page, the log key becomes `mainPage`  
- type (query) = a key from the project's links object  
- token (query or body) = user token  
- username, password (body or query) = login and password, an alternative to the token  
  
Link resolution:  
MUST:  
- If the project's links field starts with `{`, parse it as a JSON object after replacing `\"` with `"`. All values must be strings  
- Otherwise use the links field itself as the link  
- With type, use the link under that key and append `&<type>` to the log key. A type that is not in links is an error  
- Without type and with a single link, use it immediately  
- Without type and with several links, show a list of links (anchors, one per link, link texts are the keys)  
MUST: check that the chosen link is a valid URL before redirecting  
MUST: escape links and keys before putting them into HTML  
NOTE: the order of links in the list is not guaranteed  
NOTE: an empty links object produces an empty page  
  
Login logic (freejoin):  
- freejoin = 0 - redirect without login. userId in the log is taken from the token (query only) if it is valid, otherwise 0. An invalid token is not an error here  
- freejoin != 0 - redirect only after a valid username and password, or a valid token. Credentials are checked first, then the token  
- If neither succeeds, show the login form: an HTML page with `username` and `password` fields that sends a POST to the same URL, keeping the query string  
  
Log:  
MUST: write gdpsId, userId, the key and the time to the join log (joinlog table) before redirecting  
Key format: `id=<id>` or `mainPage`, plus `&<type>` when type was used  
A failed log write does not block the redirect  
  
Response:  
- Redirect: HTTP 302, Location header with the link  
- Link list and login form: HTTP 200, `text/html; charset=utf-8`  
  
Error types (plain text):  
1. id is not a number - 400, `Invalid GDPS ID`  
2. Project not found - 404, `GDPS not found`  
3. type not found in links - 404, `Unknown GDPS link type`  
4. Invalid URL - 500, `Invalid GDPS link`  
5. links is not a valid JSON object of strings - 500, `Invalid GDPS links`  
  
NOTE: `checked` is not checked, unverified projects can be joined  
  
TODO: align freejoin in the "Projects" section with the semantics above

## Project Add and Edit
The eight endpoints `send/{camp,show,pere,tele}{Add,Edit}.php` are thin wrappers over two common handlers, Add and Edit. A wrapper only sets the channel of the project, the request format, validation and response are identical for all channels  
  
### Project fields (Add and Edit)
Both handlers take the same multipart form fields  
MUST: the body is multipart/form-data, other content types fail to parse  
MUST: sanitize all text fields on the server side  
  
Fields:  
- title = project title. Required  
- language = empty for new multi-language clients, a language code for legacy single-language clients. Decides the format of description and short  
- description = project description. Required  
- - language is empty and description fields are present: repeated fields `<LANG>\t<text>` (language code, a TAB, the text). The server builds a newHelper langs dictionary from them and stores it as a JSON string. At least one valid entry is required  
- - otherwise: a single text field, stored as is (legacy field)  
- short = short description. Optional. Same two formats as description, with a separate length limit. An empty dictionary is stored as an empty string  
- img, ban = project avatar and banner. Either a file part (an image) or a text field with a URL, stored as is. If a file is sent, the text field is ignored  
- links[] = repeated field, consecutive pairs `key, value` (`discord`, `https://...`). An odd trailing element is ignored. Values are not validated as URLs  
- tags[], os[] = repeated fields with tag IDs, see the newHelper lang section  
  
How the fields are stored:  
- links: a JSON object whose quotes are escaped with a backslash, `{\"discord\":\"url\"}`. This is the legacy form that the entity renderer and /join.php undo. No links give `{}`  
- tags, os: JSON arrays of strings, exactly as sent. IDs are not validated, no tags give the JSON `null`  
- mask: a search bitmask built from the numeric values of tags and os merged together. Non-numeric values are stored but do not reach the mask  
- username: the nickname of the author, or the login name if there is no nickname. It is copied at creation and not updated by Edit  
  
NOTE: Add and Edit do not write freejoin, wiki, checked, points and hasLgbt  
  
### Project Add (common handler)
Creates a project and returns the refreshed Init response  
  
Authorization method - 3  
Method - free  
Parameters: the fields above, tokens as described in the Authorization section  
  
MUST:  
- The author of the project is the user from the tokens  
- Take gdpsId from the auto-increment of the projects table, so it is unique across all channels  
- Return the Init response with the cache bypassed, so that the new project is in the private part  
  
Images:  
- img is scaled to 256x256 WebP and saved as `imgs/customuser/i<gdpsId>.webp`  
- ban is scaled to 720x300 WebP and saved as `imgs/customuser/b<gdpsId>.webp`  
- The stored value is the full URL (`<HELPER_URL>imgs/customuser/i<gdpsId>.webp`)  
- The ID is needed for the file name, so the project is inserted first with `FILE` in img/ban, and the field is replaced after saving  
- If saving fails, the field becomes an empty string and the project stays created, the client is not told  
  
Response format:  
Init endpoint format  
  
Error types:  
1. Any of the tokens is invalid or the account is not activated - 200, `Access denied`  
2. The body can't be parsed as multipart - 500, `-1`  
3. title is empty, or description is empty in the single-language format - 200, `-27`  
4. The multi-language description is invalid or empty - 400, `-28`  
5. The multi-language short is invalid - 400, `-29`  
6. Database error on insert or on updating the pictures - 200, `-1`  
7. Invalid image - 400, `-3`  
8. Failed to build the Init response - 500, error text. The project is already created  
  
TODO: the image check runs after the insert, so an invalid image returns `-3` but leaves the project created with `FILE` in img/ban. Validate before the insert  
  
#### /server/133/send/campAdd.php
Creates a project of channel 0 (c, camp)  
  
#### /server/133/send/showAdd.php
Creates a project of channel 1 (s, show)  
  
#### /server/133/send/pereAdd.php
Creates a project of channel 2 (p, dubbing/translation). Reuses the tags of channel 1  
  
#### /server/133/send/teleAdd.php
Creates a project of channel 3 (t, telegram channel)  
  
### Project Edit (common handler)
Returns the current data of a project for the edit form, or saves new data. Returns the refreshed Init response after saving  
  
Authorization method - 2 for reading the form data, 3 for saving  
Method - free  
Parameters:  
- id (query) = gdpsId of the project  
- the fields above, tokens as described in the Authorization section  
  
Order of checks (matters for which error you get):  
1. id is a number  
2. the project exists  
3. the tokens are valid  
4. the user has access to the project  
5. the mode is picked (read or save)  
6. for saving: the account is activated  
  
MUST:  
- Let only users with access to the project through. TODO: who exactly (owner, co-owners, which permissions)  
- Pick the mode by title and description: if either of them is empty or missing (or the body can't be parsed as multipart), it is the read mode  
  
#### Read mode
Response format:  
```
[
  title,
  description,
  short,
  links,
  img,
  ban,
  tags,
  os,
  language
]
```
- description and short are returned as stored: a newHelper langs dictionary string or a legacy text, the client tells them apart  
- links is an object, as in the full entity. If the stored value is not a valid JSON object, the raw string is returned  
- tags and os are returned as stored. TODO: check whether they arrive as arrays or as JSON inside strings  
- the account does not have to be activated  
  
#### Save mode
MUST:  
- Replace all fields from the form. Omitted links, tags, os, short and language are cleared  
- Exception: img and ban keep the stored value unless a file or a non-empty text value is sent. An image can't be removed  
- Set the channel of the project from the endpoint that was called  
- Validate the images before saving anything  
- Increment the edit counter of the project on every save  
  
Images:  
- Same sizes as in Add  
- The file name contains the edit counter before the increment: `imgs/customuser/i<gdpsId>-<counter>.webp`, `b<gdpsId>-<counter>.webp`. Every edit gets a new URL, so caches don't hold the old picture  
- Old files are not removed  
- If saving a file fails, the response is 500 `-3`. The text fields are already saved by that time, the pictures are unchanged  
  
Response format:  
Init endpoint format  
  
Error types:  
1. id is not a number - 400, `-7`  
2. The project does not exist - 404, `-6`  
3. Any of the tokens is invalid - 200, `Access denied`  
4. The user has no access to the project - 200, `-2`  
5. The account is not activated (save mode) - 200, `Access denied`  
6. Multi-language description is invalid or empty - 400, `-28`  
7. Multi-language short is invalid - 400, `-29`  
8. Description is empty in the single-language format - 200, `-27`. Practically unreachable, an empty description switches the handler to the read mode  
9. Invalid image - 400, `-3`  
10. Database error on saving or on updating the pictures - 200, `-1`  
11. Failed to save an image file - 500, `-3`  
12. Failed to build the Init response - 500, error text. The data is already saved  
  
NOTE: errors 1 and 2 are returned before the tokens are checked  
  
TODO: the channel is rewritten from the endpoint without checking the current channel, so calling the Edit of another channel moves the project to it. Is it intended  
  
#### /server/133/send/campEdit.php
Sets the channel of the project to 0 (c, camp)  
  
#### /server/133/send/showEdit.php
Sets the channel of the project to 1 (s, show)  
  
#### /server/133/send/pereEdit.php
Sets the channel of the project to 2 (p, dubbing/translation)  
  
#### /server/133/send/teleEdit.php
Sets the channel of the project to 3 (t, telegram channel)  
  


# TODO: endpoints
  
Checklist for each endpoint (copy into the description):  
- authorization method (0-3)  
- method (free, GET or POST)  
- parameters (body or query)  
- response format  
- MUST/SHOULD  
- error table  
- mark "canonical / openGo only / Legacy PHP only"  
  
Legend: [POST] - openGo accepts only POST, no mark - free method  
All paths are relative to /server/133/  
  
## Already described in the protocol (nothing to do)  
- send/like.php [POST] (Likes)  
- send/dislike.php [POST] (Likes)  
- likesT.php (Likes)  
- loginT.php (Init-endpoint)  
- send/comment.php [POST] (Comments)  
- content/fetchComms.php (Comments)  
- send/commentModify.php [POST] (Comments)  
- delete/comment.php (Comments)  
  
## Meta-domains

### Authorization
- user/login.php [POST]: TODO response in the Init endpoint format, token in the token field, degradations do not apply  
- user/logout.php [POST]  
- user/register.php [POST]: TODO response in the Init endpoint format, same as login  
- challenge.php: Altcha captcha, TODO challenge format, where it is required (registration? login? comments?)  
TODO: authorization methods 0-3 are described but not tied to the endpoint list. Add a summary table "endpoint - authorization method"  

### Search
- search/new.php: main project search, TODO parameters (query, channel, tags, os, page, sorting), response format (short entity "project")  
- wiki/getWikis.php: wiki search (lives in zSearch.go in the code)  
- vacans/getAll.php: vacancy search (lives in zSearch.go in the code)  
TODO: decide whether wiki/getWikis and vacans/getAll stay in Search or move to Wiki/Vacancies, leaving a link in Search  

## Users
  
### Profile (new section, not in the table of contents, I suggest adding it)
- user/getAccInfo.php [POST]  
- user/setNickname.php  
- user/setSocials.php  
- user/setResume.php: account for R3, resume is empty by default and not rendered since 0.99  
  
### Public profile (also a new section)
- content/getUser.php  
- content/getAddedCamps.php  
- content/getAddedShows.php  
- content/getAddedPeres.php  
- content/getAddedTeles.php  
- content/getUserGuides.php  
TODO: is the "short project entity" format shared with search or separate  
TODO: pagination  

### Portfolio (R3)
- user/works.php  
- user/workAdd.php  
- user/workEdit.php  
- user/workDelete.php [POST]  
- user/workVerify.php [POST]  
TODO: "work" entity (fields, format)  
TODO: who verifies and how (workVerify), statuses  
TODO: relation to resume  

### Devices
- user/devices.php  
- user/removeDevice.php  
- send/deviceAdd.php [POST]  
TODO: device token (device-static), lifetime, list format  

### Notifications + Pushes
- content/getAlarms.php  
- content/getAlarm.php  
- send/deleteAlarm.php  
- send/writeAlarm.php  
- sub.php [POST]: push subscription (zPushs.go), not to be confused with project subscriptions  
TODO: "alarm" entity, types, read/unread  
TODO: push subscription format (endpoint, keys)  

## Projects
  
### Entities and creation (the section already describes entities, but has no endpoints)
- content/camp.php: opening the project page (full entity), despite the name it most likely returns all channels. TODO: verify  
- send/campAdd.php  
- send/showAdd.php  
- send/pereAdd.php  
- send/teleAdd.php  
- send/campEdit.php  
- send/showEdit.php  
- send/pereEdit.php  
- send/teleEdit.php  
- send/bump.php: bumping a project. TODO: limits, permissions  
- reportGdps.php [POST]: project report. TODO: decide whether a separate "Reports" section is needed  
TODO: do Add/Edit have different fields per channel? Describe the common part once and the differences in a table  
TODO: there is no project deletion endpoint in the list. Does it not exist, or is it not in the protocol?  
TODO: img/ban upload goes through /loader (see the "Uploader" section)  

### News
- send/newsPost.php [POST]  
- send/newsModify.php [POST]  
- delete/newsPost.php  
- content/newsAll.php: global news (zGlobalNews.go)  
- content/news.php: project news  
- content/newsC.php: a single news item  
TODO: "news" entity, binding to gdpsId  
TODO: permissions (project author, co-owners)  

### Vacancies
- vacans/getAll.php (see Search)  
- vacans/get.php  
- content/vacsC.php: a single vacancy  
- send/vacsAdd.php  
- send/vacsEdit.php  
- vacans/edit.php: preloading data for the edit form (VacsEditPre)  
- vacans/removeVac.php  
- vacans/apply.php: application  
- vacans/removeApl.php: withdrawing an application  
- vacans/applies.php: list of applications  
TODO: "vacancy" and "application" entities, statuses  

### Subscriptions (R2)
- gdps/sub.php  
- gdps/unsub.php  
- gdps/subs.php  
TODO: relation to likesT.php, the subs key  
TODO: how subscriptions produce pushes (the code lives in zAlarms.go)  

### Co-owners
- content/getOwners.php  
- send/permAdd.php  
- send/perm.php: judging by the PermRemove handler, this is permission removal. TODO: verify, rename in the description  
- content/getJoinLog.php  
TODO: "permission" entity and the list of permissions  
TODO: freejoin and the join log  

## Wiki

### Wiki and styling
- wiki/getWiki.php: the same handler as getGuides in the code (GuidesHandler). TODO: do the responses differ  
- send/newWiki.php  
- send/editWiki.php  
- wiki/colors.php  
- wiki/setMainWiki.php  
- wiki/getWikis.php (see Search)  

### Link to projects
- search/conntectWiki.php: typo in the path ("conntect"), keep as is, mark it in the protocol  
- search/conntectContent.php: same  
TODO: what exactly each of the two links  
TODO: does the link support "many projects - one wiki"  

### Articles ("guides")
- wiki/getGuides.php  
- wiki/getGuide.php  
- content/getGuidesAdmin.php  
- send/newGuide.php  
- send/editGuide.php  
TODO: "article" entity  
TODO: explicitly bind like type 7 and comment channel 2  

### Article tags (planned to be renamed to addresses)
- wiki/setWikiTag.php  
TODO: decide whether the rename affects the path too  

### Templates
- wiki/templateGet.php  
- wiki/templatesGet.php  
- wiki/templateSave.php  
- wiki/templateDelete.php  
TODO: template format, restrictions  

### Forums
- forum/create.php  
- forum/createPost.php  
- send/forumPost.php: alias to the same handler (ForumCreatePost). Describe as an alias, once  
- forum/getPost.php  
- forum/getPosts.php  
TODO: "forum" and "post" entities, like type 9, comment channel 4  

### Comic reader
- no separate endpoints in the list  
TODO: find out how it is served (most likely inside wiki/getGuide.php), and either describe the format or remove it from the domain list  

### Wiki co-owners
- no separate endpoints visible, possibly uses content/getOwners.php and send/permAdd.php  
TODO: check whether this is the same system as for projects (the domains section says they will differ in the future)  

## Uploader (not a domain, but worth a separate note)
- /loader: TODO purpose, format, limits  
- /join.php: TODO purpose (GDPSjoin handler, judging by the name, joining a project)  
- wiki/filesGet.php, wiki/filesSend.php, search/deleteWikiFiles.php: stubs (DropPhp) in openGo. TODO: decide whether to document as "not implemented in openGo"  
TODO: note that the uploader is not tied to a revision (this is already written in the "Explaining the cringe" section)  

## Outside the protocol (one line in the document is enough, or nothing)
- / (IndexParser)  
- wordleRU.php, wordleEN.php  
- !newTakeAll.php, Aaction.php: admin  
- vless.php: DropPhp stub  
TODO: decide what to do with the admin endpoints (document as internal or don't mention at all)  

## Divergences I noticed (worth moving into an "openGo divergences" block)
- like, dislike, comment, commentModify are POST-only in openGo, while the protocol says "free method"  
- fetchComms is specified as GET in the protocol, in openGo the method is free  
- loginT.php has a handler named FreeBSDcompile, either a joke or a forgotten name  
- The paths search/conntectWiki and search/conntectContent contain a typo  
- Wiki file deletion lives in search/ (you left a comment about it in the code yourself)  
- wiki/getWiki.php and wiki/getGuides.php lead to one handler, and so do send/forumPost.php and forum/createPost.php  
- There is no endpoint for deleting a project, nor for updating/deleting forums, articles and wikis  

## Order (by dependencies, as before)
1. Entities User, News, Vacancy (Project already exists)  
2. Authorization, login/register/logout, loginT, Devices  
3. Projects (endpoints), News, Subscriptions  
4. Search  
5. Co-owners, Vacancies  
6. Portfolio (R3), Profile  
7. Notifications + Pushes  
8. Wiki, Articles, Tags, Templates, Forums  

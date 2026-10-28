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

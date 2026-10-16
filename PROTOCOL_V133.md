# Object Hub Protocol V133 R3
  
WARNING!!! THIS DOCUMENT IS NOT FINISHED, DON'T BE SURPRISED BY BROKEN LINKS  
Version 133 of Object Hub protocol, new revisions are backward compatible for target clients  
First client version on this protocol - 0.97.4  
This will not be a description of backend operation, but a description of the PROTOCOL, just like X11 protocol, which doesn't know what a video card is  
  
Here also may be detailed description of protocol divergences from Object Hub openGo project  
  
# Contents  

- Revisions
- - R0
- - R1
- - R2
- - R3
- Philosophy
- History
- - Historically-grown problems
- Functional domains
- Meta domains
- - Authorization
- - Init endpoint
- - Likes
- - Comments
  

# Revisions
  
A revision is a backward-compatible extension of the protocol  
Revisions are designated as R<N>  
  
Adding new functionality to the backend inevitably leads to an increase in the revision number  
If backward compatibility is violated in any part of the protocol when creating a new revision - this is already a new version, not a revision  
  
## R0
Canonical revision of version 133, released together with Object Hub 0.97.4 NODE\_BETA1  
  
## R1
Released with release 0.97.4 NODE\_BETA3  
Extensions:  
- Support for sending user&device tokens via body, next to headers. Added to preserve compatibility with 132nd version of protocol  
- likesT.php endpoint - hashset returning all all records of user's like status, used to verify like placement on client (version 133 initially was not supposed to verify likes on server, but I forgot to add client cache and had to extend revision)  
  
## R2
Released with Object Hub 0.98 release  
Extensions:  
- Subscriptions to projects  
- Many endpoints received error codes, they still do not have clear recording in protocol  
  
## R3
Will be completed with Object Hub 0.99 release, its temporary variant is available already with Object Hub 0.98.2  
Extensions:  
- Entire portfolio infrastructure  
Changes:  
- Field resume in profile will become empty by default because since 0.99 it is not rendered, instead it will be by default in portfolio  
- - Note that the field will be empty, and not deleted. This is done to formally preserve compatibility with old clients

# Philosophy
In the history of Object Hub/GDPS Helper, client and server developed in parallel and very coordinated, with the goal of maximally minimizing traffic, number of API requests, number of bytes in response, and ultimately your server  
For this reason the entire protocol will look very bad and strange, it was built strictly for requirements of official Object Hub client, and not to be a REST API  
And the goal of the protocol's existence as such - to optimize your backend and network consumption of Object Hub API (yes, I voiced this, repetition is mother of learning)
  

# History
Object Hub is a fork of GDPS Helper, it had its own problems and peculiarities in protocol  
The very first version of Object Hub protocol was GDPS Helper 1.7 implementing protocol V64 (which was then still very unstable)  
Protocol partially settled by GDPS Helper 1.8 - domains were brought out completely, first version of Legacy PHP backend was ready  
After forking beta version of GDPS Helper 1.8 I in Object Hub started breaking protocol "for beauty" and "for convenience", breaking some part of aggressive optimizations (not everything, as it seems at first)  
Protocol relatively settled only by version 104, all Object Hub clients from build 104 (client version 0.94) will most likely work without particular complaints, or with minimal mokey-patches  
Fork of GDPS Helper Legacy PHP, having survived many object hub specific updates, became basis of Object Hub Protocol V133 R0  
  
## Historically-grown problems
- Protocol is insufficiently unified
- - Some entities respond with objects, others with arrays
- Error formats differ from endpoint to endpoint, therefore each endpoint will have its own error table
- - Though in many places response is sufficiently unified - 200, `Access denied`
- The above problems will not be fixed for sake of preserving compatibility with legacy clients  
- Starting with V133R1 protocol took up preserving compatibility with much older versions, because of this on a huge number of endpoints there is a "free method", because all Object Hub clients from 0.97.33 and older either didn't send token and method was GET, or sent token in body and method became POST
  

# Functional domains  
## Meta domains
- Authorization
- Init endpoint
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
- Tags (need to rename to addresses) of articles
- Articles
- Comic reader
- Templates
  
## Explaining the cringe:  
- Co-owner systems of GDPSs and Wiki will be different in future, therefore this is not a meta domain
- The loader is just a loader, it should not be a domain because it is not "business logic", also it is not tied to a specific revision and can internally change as I need, although I don't see much sense in this right now

# Meta domains
  
These are entities that leak through all domains and other entities of Object Hub protocol. Still is the most controversial functional domain  
  
Likes, comments, and search are domains that by the field determining type implement their internals (likes place a like, search searches projects and not everything, comment is placed under news and not somewhere unclear)  

## Authorization
Actually this is not even a domain, but a very important part of functionality of half of API endpoints  
Authorization checks user rights to a particular entity and other execution conditions, and if check fails issues an error (usually response "Access denied" without additional headers)  
Authorization is divided into 4 methods:
0. Public data
  Here there are no rights checks, this data is available even to a bot-scanner that for some reason decided to scan port 443 at address `/server/133/search/new.php` of your server  
1. User data
  This authorization method usually checks only correctness of user token, occurs on Read endpoints with private data  
  For example - list of devices, local database of likes  
2. Updating private data
  This authorization method verifies user token and device token, ignoring account activation status, occurs on all private Write endpoints  
  For example - Deleting device, changing password on account  
3. Updating public data
  This authorization method verifies user token and device token, checking whether account activation status is not zero, occurs on all public Write endpoints  
  For example - placing like, sending comment, adding project  
Existence of method 1 is justified by optimizing database load
  
### Ways to send tokens
MUST: rights check uses tokens sent by different ways identically  
SHOULD: first tokens from headers are checked  
  
1. headers  
  User token goes in `user-token` header, device token in `device-static` header  
2. body  
  User token goes in `token` field in request body, device token goes in `device` field in request body  

## Init endpoint
Endpoint responsible for initialization of official Object Hub client, usually contains cache of private data and a bit of public  
Historically is response not only for `loginT.php`, but also for many other endpoints, and sometimes with strange semantics  
  
Authorization method - 2. Here exactly this authorization method is used to block account on client under ordinary circumstances, see error types  
Method - free  
Parameters absent  
  
MUST:  
- Ignore device check and return user token in field token inside Current_user when using Init function on registration and login  
- Not touch public part of response despite all authorization errors  
SHOULD:  
- When using cache, separate public and private data into 2 different caches, so as not to give another user someone else's data  
- When using guest user in response, not cache it but store directly in your backend cache  
  
Response format:

# Object Hub Protocol V133 R3

**WARNING! THIS DOCUMENT IS INCOMPLETE. DO NOT BE SURPRISED BY BROKEN LINKS.**

Version 133 of the Object Hub protocol. New revisions are backward-compatible with the clients they target.

The first client version using this protocol is 0.97.4.

This document describes the **PROTOCOL**, not how the backend works—just as the X11 protocol does not know what a graphics card is.

This document may also describe protocol differences specific to the Object Hub openGo project.

# Table of Contents

- Revisions
  - R0
  - R1
  - R2
  - R3
- Philosophy
- History
  - Historically Accumulated Problems
- Functional Domains
- Meta-Domains
  - Authentication
  - Init Endpoint
  - Likes
  - Comments

# Revisions

A revision is a backward-compatible extension of the protocol.

Revisions are designated as `R<N>`.

Adding new functionality to the backend inevitably increases the revision number.

If backward compatibility is broken in any part of the protocol when creating a revision, it is a new version, not a revision.

## R0

The canonical revision of version 133, released with Object Hub 0.97.4 NODE_BETA1.

## R1

Released with version 0.97.4 NODE_BETA3.

Extensions:
- Support for sending user and device tokens in the request body as well as in headers. This was added to preserve compatibility with version 132 of the protocol.
- The `likesT.php` endpoint: a hash set containing all records of the user's like statuses. It is used to synchronize the like state on the client. Version 133 was originally not supposed to synchronize likes with the server, but I forgot to add a client-side cache, so the revision had to be extended.

## R2

Released with Object Hub 0.98.

Extensions:
- Project subscriptions.
- Many endpoints gained error codes, although these are still not formally specified in the protocol.

## R3

Will be completed with Object Hub 0.99. A provisional version has been available since Object Hub 0.98.2.

Extensions:
- The entire portfolio infrastructure.

Changes:
- The `resume` field in the profile will be empty by default because it will no longer be rendered starting with 0.99. Instead, it will be included in the portfolio by default.
- Note that the field will be empty, not removed. This formally preserves compatibility with older clients.

# Philosophy

Throughout the history of Object Hub/GDPS Helper, the client and server evolved in parallel and in close coordination, with the goal of minimizing traffic, the number of API requests, the number of bytes in responses, and, ultimately, the load on your server.

For this reason, the protocol may look very ugly and strange. It was designed strictly around the requirements of the official Object Hub client, not to serve as a REST API.

The protocol's purpose is to optimize your backend and Object Hub API network usage. Yes, I've said this before; repetition is the mother of learning.

# History

Object Hub is a fork of GDPS Helper, which had its own protocol problems and peculiarities.

The first version of the Object Hub protocol was GDPS Helper 1.7, which implemented protocol V64 (which was still very unstable at the time).

By GDPS Helper 1.8, the protocol had partially settled down: domains had been fully separated, and the first Legacy PHP backend was ready.

After forking the beta version of GDPS Helper 1.8, I started breaking the protocol in Object Hub for the sake of “beauty” and “convenience”, undoing some of its aggressive optimizations (not all of them, despite what it might seem at first).

The protocol did not really stabilize until version 104. All Object Hub clients from build 104 onward (client version 0.94) will most likely work without significant issues, or with only minimal monkey patches.

The GDPS Helper Legacy PHP fork, which survived many Object Hub-specific updates, became the foundation of Object Hub Protocol V133 R0.

## Historically Accumulated Problems

- The protocol is not sufficiently consistent.
  - Some entities are returned as objects, while others are returned as arrays.
- Error formats vary from endpoint to endpoint, so each endpoint has its own error table.
  - In many cases, however, responses are fairly consistent: `200`, `Access denied`.
- These problems will not be fixed, in order to preserve compatibility with outdated clients.
- Starting with V133 R1, the protocol began preserving compatibility with much older versions. As a result, many endpoints use a “free method” because Object Hub clients version 0.97.33 and older either sent no token and used the GET method, or sent the token in the body and therefore used POST.

# Functional Domains

## Meta-Domains

- Authentication
- Init endpoint
- Likes
- Comments
- Search

## Users

- Portfolio
- Devices
- Notifications + Push notifications

## Projects

- News
- Jobs
- Subscriptions (directly trigger push notifications)
- Co-owners

## Wiki

- Forums
- Co-owners
- Tags (should be renamed to article paths)
- Articles
- Comic reader
- Templates

## Explaining the Weirdness

- The co-owner systems for GDPSes and the Wiki will be different in the future, so this is not a meta-domain.
- The loader is just a loader. It should not be a domain because it is not “business logic”. It is also not tied to a specific revision and can be changed internally whenever needed, although I do not see much reason to do so right now.

# Meta-Domains

These are entities that cut across all domains and other entities in the Object Hub protocol. This is still the most controversial functional domain.

Likes, comments, and search are domains whose internal behavior is determined by a field specifying the type: likes apply a like, search searches for projects rather than everything indiscriminately, and a comment is posted under a news item rather than somewhere unspecified.

## Authentication

Strictly speaking, this is not even a domain, but it is an important part of the functionality of many API endpoints.

Authentication checks the user's permissions for a given entity and other execution conditions. If a check fails, it returns an error (usually the response `Access denied`, without additional headers).

There are four authentication methods:

0. **Public data**  
   No permission checks are performed. This data is available even to a scanning bot that, for whatever reason, decided to scan port 443 at `/server/133/search/new.php` on your server.

1. **User data**  
   This method usually checks only whether the user token is valid. It is used by read endpoints that return private data.  
   Examples: device list, local like-status database.

2. **Updating private data**  
   This method checks the user token and device token, while ignoring the account's activation status. It is used by all private write endpoints.  
   Examples: deleting a device, changing the account password.

3. **Updating public data**  
   This method checks the user token and device token and verifies that the account's activation status is not zero. It is used by all public write endpoints.  
   Examples: liking something, posting a comment, adding a project.

The existence of method 1 is justified by the need to reduce database load.

### Ways to Send Tokens

**MUST:** Permission checks must treat tokens sent through the different supported methods identically.

**SHOULD:** Check tokens from headers first.

1. **Headers**  
   The user token is sent in the `user-token` header, and the device token in the `device-static` header.

2. **Body**  
   The user token is sent in the `token` field of the request body, and the device token in the `device` field.

## Init Endpoint

The endpoint responsible for initializing the official Object Hub client. It usually contains a cache of private data and some public data.

Historically, it has served as a response not only for `loginT.php`, but also for many other endpoints, sometimes with strange semantics.

Authentication method: **2**. This method is used here specifically to lock the account on the client under normal circumstances; see the error types.

Method: **free**.

Parameters: none.

**MUST:**
- Skip device validation and return the user token in the `token` field inside `Current_user` when the Init function is used during registration and login.
- Do not modify the public part of the response, regardless of authentication errors.

**SHOULD:**
- If caching is used, keep public and private data in two separate caches to avoid returning one user's data to another user.
- If a guest user is included in the response, do not cache that user separately; store it directly in your backend's cache.

Response format:

```json
[
  // Start of private cache
  Current_user,
  [
    Projects_owned_by_user{},
    Wikis_owned_by_user{}
  ],
  // End of private cache; start of public cache
  Project_cache_for_home_page{},
  News_cache_for_home_page[],
  // End of public cache
]
```

Note that wherever an object is expected, the response is an object whose keys are:

```json
[
  // Private part of the response
  User,
  [
    {
      "c1": Project,
      "p3": Project,
      "t4": Project
    },
    {
      "w1": Wiki,
      "w2": Wiki,
      "w3": Wiki
    }
  ],
  // Public part of the response
  {
    "c1": Project,
    "s2": Project
  },
  [
    News,
    News
  ]
]
```

To understand what the letters in the response mean, see the “Projects” section.

### Error Format

Usually, if an authentication error occurs on the server side (for example, because the device token has expired), the client receives a “guest user”: instead of the current user, the response contains a standard placeholder whose `token` field literally contains the string `"false"` (not a boolean). The array of the user's projects and wikis will be empty: `[{},{}]`.

If no tokens are provided at all (the user does not have an account), the guest user will not have a `token` field in the response.

Degradation cases:
1. Expired tokens — `200`, guest user + `token:"false"`.
2. No tokens provided — `200`, guest user with no `token` field.
3. The city and country cannot be determined from the IP address — the `cityData` field falls back to `["Unknown","Unknown"]`.

## Likes

A domain responsible for checking likes on a specific entity item.

**MUST:** Prevent the same like action from being submitted more than once.

**SHOULD NOT:** Recalculate the number of likes by scanning the status table on every action (including simply opening a project).

## Object Hub Like Types

This is a separate enum-like namespace:

- `0` = Projects
- `1` = Comments on projects
- `2` = News
- `3` = Dead field; previously used for GDPS Helper texture packs, before they were removed in GDPS Helper 1.9
- `4` = Dead field; previously used for comments on GDPS Helper texture packs, before they were removed in GDPS Helper 1.9
- `5` = Comments on news
- `6` = Comments on Wiki articles
- `7` = Wiki articles
- `8` = Wikis
- `9` = Forum posts
- `10` = Comments on forum posts
- `11` = Jobs
- `12` = Comments on jobs

## Like Rollback

Each entity object stores its own list of likes and dislikes separately. The record of a user's like status is stored in a hashmap/table indexed by identifiers.

If a record for an existing like is found and an attempt is made to set a new rating, the corresponding count is adjusted in the opposite direction based on the existing record in the hashmap. The like-status record is then deleted from the table.

## Endpoints

### `/server/133/send/like.php`

Increments the like count.

Authentication method: **3**.

Method: **free**.

Parameters (either in the body or query string):
- `ide` = Entity ID
- `type` = Entity type. For example, in Object Hub, `type=0` means a project and `type=1` means a comment on a project.

Response format:

```json
[
  Like_count,
  Dislike_count
]
```

openGo error types (not part of the canonical protocol):
1. Either token is invalid — `200`, `Access denied`.
2. Invalid `ide` — `500`, `invalid id`.
3. Invalid `type` — `500`, `invalid type`.
4. `type` is not in the registered channel table — `500`, `Unknown like type`.

### `/server/133/send/dislike.php`

Increments the dislike count.

Authentication method: **3**.

Method: **free**.

Parameters (either in the body or query string):
- `ide` = Entity ID
- `type` = Entity type. For example, in Object Hub, `type=0` means a project and `type=1` means a comment on a project.

Response format:

```json
[
  Like_count,
  Dislike_count
]
```

openGo error types (not part of the canonical protocol):
1. Either token is invalid — `200`, `Access denied`.
2. Invalid `ide` — `500`, `invalid id`.
3. Invalid `type` — `500`, `invalid type`.
4. `type` is not in the registered channel table — `500`, `Unknown like type`.

### `/server/133/likesT.php`

A hash-set cache used to determine how like buttons should be colored in the Object Hub client.

It was made a separate endpoint from `loginT.php` to balance the load from TCP handshakes.

Authentication method: **1**; in practice, openGo implements method **2**.

Method: **free**.

Parameters: none.

Response format:

```json
{
  "p": "projects",
  "c": "comments",
  "n": "news",
  "g": "guides" - Wiki articles,
  "w": "wikis",
  "f": "forum posts",
  "v": "jobs",
  "subs": "project subscriptions, which use the like infrastructure solely because their storage mechanisms are similar; introduced in Revision 2"
}
```

Error types:
1. Either token is invalid — `200`, `Access denied`.
2. The account does not exist — `200`, `Access denied`.

## Comments

A cross-cutting domain responsible for storing and handling comments.

### The “Comment” Entity

```json
[
  ID,
  Author_nickname,
  Text,
  Author_UserID,
  Author_role,
  [
    Like_count,
    Dislike_count
  ],
  Unix_timestamp,
  0
]
```

The final `0` is a dead field. It previously indicated whether the user had liked the comment; this behavior remains in Legacy PHP.

### Standard Comment Getter

This is an internal backend method (or function) that must strictly follow this positional argument format:

`Function(Channel, ID, Page)`

In Legacy PHP, the third argument is `userId` and the fourth is `Page`.

This function must return a list of 11 comments, if possible. The offset is calculated as `OFFSET = page * 10`.

This is because the client checks whether an 11th item exists. If it does, the client resets the rendering and inserts a “Show more” button. To save bandwidth, you do not need to return a full 11th comment; you can simply put `0` in its place if you choose to return an array.

Response formats accepted by the client:

```json
[
  Comment,
  Comment,
  ...
]
```

```json
{
  "c<ID>": Comment,
  "c<ID>": Comment,
  ...
}
```

For an empty result, return `[]` or `{}`; the client makes no distinction between them.

### Object Hub Comment Channels

This is a separate enum-like namespace:

- `0` = Projects; like channel = `1`
- `1` = Dead field; previously used for GDPS Helper texture packs, before they were removed in GDPS Helper 1.9; like channel = `4`
- `2` = “Guides” (Wiki articles); like channel = `6`
- `3` = News; like channel = `5`
- `4` = Forum posts; like channel = `10`
- `5` = Jobs; like channel = `12`

### `/server/133/send/comment.php`

Posts a comment under an entity and returns the current list of comments on that entity's first page.

Authentication method: **3**.

Method: **free**.

Parameters (either in the body or query string):
- `ide` = ID of the entity the comment is posted under
- `type` = Comment channel
- `text` = Comment text

**MUST:**
- Sanitize `text` on the server before checking its length.
- Do not create a comment if the sanitized `text` is shorter than 10 **bytes** (UTF-8), not characters. Cyrillic characters occupy 2 bytes each.
- Convert an invalid `type` (not a number) to channel `0`.
- Convert `type=1` (the dead channel) to channel `0`.
- Return the comment list with status `200` even if the comment was not created because of rate limiting. The client does not know that the comment was discarded.

**SHOULD:**
- Rate-limit comments per user. openGo: 1 comment every 5 minutes; Legacy PHP: 1 comment every 10 minutes.
- Avoid relying on a fixed list of channels. An unknown channel is not an error: the comment should go to channel `0`, so that a new channel cannot be “forgotten during registration” and cause an error.

Response format:

The Standard Comment Getter.

Error types:
1. Either token is invalid, or the account is not activated — `200`, `Access denied`.
2. Invalid `ide` (not a number or `0`) — `400`, `Invalid id` (openGo-specific).
3. Text shorter than 10 bytes after sanitization — `200`, `-4`.
4. Internal error while writing or reading comments — `500`, error text (openGo-specific).

### `/server/133/content/fetchComms.php`

Returns a page of comments under an entity. The client uses it to load subsequent pages.

Authentication method: **0**.

Method: **GET**.

Query parameters:
- `type` = Comment channel
- `id` = ID of the entity the comments belong to
- `page` = Page number, starting at `0`

**MUST:**
- Convert invalid values (non-numeric or missing) for `type`, `id`, and `page` to `0`.

Response format:

The Standard Comment Getter.

openGo error types (not part of the canonical protocol):
1. Internal error while reading comments — `500`, `Failed to get comments: <error text>`.
2. Internal error while serializing the response — `500`, `Failed to encode comments: <error text>`.

### `/server/133/send/commentModify.php`

Modifies the text of an existing comment and returns the new text.

Authentication method: **3**.

Method: **free**.

Parameters (either in the body or query string):
- `id` = ID of the comment itself (not the entity it belongs to)
- `text` = New comment text

**MUST:**
- Allow a comment to be edited only by its author or a moderator (a user with nonzero `priority`). Moderators can edit any comment.
- Sanitize `text` on the server before saving.
- **Do not** update the comment's `Unix_timestamp` when it is edited.
- On success, return the raw `text` from the request with status `200`. The client displays this response in place of the old text.

Response format:

```text
New comment text, as a plain string
```

Error types:
1. Either token is invalid — `200`, `Access denied`.
2. Invalid `id` (not a number) — `400`, `Invalid id` (openGo-specific).
3. Account is not activated — `200`, empty response.
4. Text shorter than 10 bytes before sanitization — `200`, `-4`.
5. The comment belongs to another user (and the current user is not a moderator), or the comment does not exist — `200`, `-3`.
6. Internal error while reading or writing the comment — `500`, error text (openGo-specific).

### `/server/133/delete/comment.php`

Deletes a comment along with its likes and returns the ID of the deleted comment.

Authentication method: **3**.

The HTTP method may vary for the tokens, but `ide` **MUST** be passed in the query string.

Parameters:
- `ide` (query) = ID of the comment itself (not the entity it belongs to)
- `type` (query) = Comment channel. The client sends this, but openGo ignores it; the channel is taken from the comment itself.

**MUST:**
- Allow a comment to be deleted only by its author or a moderator (a user with nonzero `priority`). Moderators can delete any comment.
- Delete the like/dislike records for the comment along with the comment itself.

**SHOULD:**
- Determine the channel from the comment itself rather than from the `type` parameter. The client may send an incorrect `type`, which could cause likes to be deleted from the wrong namespace.

Response format:

```text
ID of the deleted comment, as a plain string
```

Error types:
1. Either token is invalid — `200`, `Access denied`.
2. The comment belongs to another user, and the current user is not a moderator — `200`, `-1`.
3. Invalid `ide` (not a number) — `400`, `Invalid id` (openGo-specific).
4. Internal error while reading or deleting the comment — `500`, error text (openGo-specific).

package blueskyapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Preloading/MastodonTwitterAPI/bridge"
)

type AuthResponse struct {
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
	DID        string `json:"did"`
}

type AuthRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type Author struct {
	DID            string `json:"did"`
	Handle         string `json:"handle"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Avatar         string `json:"avatar"`
	Banner         string `json:"banner"`
	FollowersCount int    `json:"followersCount"`
	FollowsCount   int    `json:"followsCount"`
	PostsCount     int    `json:"postsCount"`
	IndexedAt      string `json:"indexedAt"`
	CreatedAt      string `json:"createdAt"`
	Associated     struct {
		Lists        int       `json:"lists"`
		FeedGens     int       `json:"feedgens"`
		StarterPacks int       `json:"starterPacks"`
		Labeler      bool      `json:"labeler"`
		CreatedAt    time.Time `json:"created_at"`
	}
}

type PostRecord struct {
	Type      string    `json:"$type"`
	CreatedAt time.Time `json:"createdAt"`
	Embed     Embed     `json:"embed"`
	Facets    []Facet   `json:"facets"`
	Langs     []string  `json:"langs"`
	Text      string    `json:"text"`
}

// Specifically for reposts
type PostReason struct {
	Type      string    `json:"$type"`
	By        Author    `json:"by"`
	IndexedAt time.Time `json:"indexedAt"`
}

type Embed struct {
	Type   string  `json:"$type"`
	Images []Image `json:"images"`
}

type Image struct {
	Alt         string      `json:"alt"`
	AspectRatio AspectRatio `json:"aspectRatio"`
	Image       Blob        `json:"image"`
}

type AspectRatio struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type Blob struct {
	Type     string `json:"$type"`
	Ref      Ref    `json:"ref"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}

type Ref struct {
	Link string `json:"$link"`
}

type Facet struct {
	Features []Feature `json:"features"`
	Index    Index     `json:"index"`
}

type Feature struct {
	Type string `json:"$type"`
	Tag  string `json:"tag"`
	Did  string `json:"did"`
}

type Index struct {
	ByteEnd   int `json:"byteEnd"`
	ByteStart int `json:"byteStart"`
}

type PostViewer struct {
	Repost            *string `json:"repost"`
	Like              *string `json:"like"` // Can someone please tell me why this is a string.
	Muted             bool    `json:"muted"`
	BlockedBy         bool    `json:"blockedBy"`
	ThreadMute        bool    `json:"threadMute"`
	ReplyDisabled     bool    `json:"replyDisabled"`
	EmbeddingDisabled bool    `json:"embeddingDisabled"`
	Pinned            bool    `json:"pinned"`
}
type Post struct {
	Subject
	Author Author     `json:"author"`
	Record PostRecord `json:"record"`
	// Embed  Embed      `json:"embed"`
	ReplyCount  int        `json:"replyCount"`
	RepostCount int        `json:"repostCount"`
	LikeCount   int        `json:"likeCount"`
	QuoteCount  int        `json:"quoteCount"`
	IndexedAt   string     `json:"indexedAt"`
	Viewer      PostViewer `json:"viewer"`
}

type Feed struct {
	Post  Post `json:"post"`
	Reply struct {
		Root   Post `json:"root"`
		Parent Post `json:"parent"`
	} `json:"reply"`
	Reason      *PostReason `json:"reason"`
	FeedContext string      `json:"feedContext"`
}

type Timeline struct {
	Feed   []Feed `json:"feed"`
	Cursor string `json:"cursor"`
}

type Thread struct {
	Type    string `json:"$type"`
	Post    Post   `json:"post"`
	Parent  Post   `json:"parent"`
	Replies []Post `json:"replies"`
}

// This is solely for the purpose of unmarshalling the response from the API
type ThreadRoot struct {
	Thread Thread `json:"thread"`
}

// Reposting/Retweeting
type CreateRecordPayload struct {
	Collection string       `json:"collection"`
	Repo       string       `json:"repo"`
	Record     RepostRecord `json:"record"`
}

type DeleteRecordPayload struct {
	Collection string `json:"collection"`
	Repo       string `json:"repo"`
	RKey       string `json:"rkey"`
}

type RepostRecord struct {
	Type      string  `json:"$type"`
	CreatedAt string  `json:"createdAt"`
	Subject   Subject `json:"subject"`
}

type Subject struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type Commit struct {
	CID string `json:"cid"`
	Rev string `json:"rev"`
}

type CreateRecordResult struct {
	Subject
	Commit           Commit `json:"commit"`
	ValidationStatus string `json:"validationStatus"`
}

type RepostedBy struct {
	Subject
	Cursor     string   `json:"cursor"`
	RepostedBy []Author `json:"repostedBy"`
}
type Likes struct {
	Subject
	Cursor string           `json:"cursor"`
	Likes  []ItemByWithDate `json:"likes"`
}

type ItemByWithDate struct {
	IndexedAt time.Time `json:"indexedAt"`
	CreatedAt time.Time `json:"createdAt"`
	Actor     Author    `json:"actor"`
}

func Authenticate(username, password string) (*AuthResponse, error) {
	url := "https://bsky.social/xrpc/com.atproto.server.createSession"

	authReq := AuthRequest{
		Identifier: username,
		Password:   password,
	}

	reqBody, err := json.Marshal(authReq)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("authentication failed")
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// TODO: This looks like it's a bsky.social specific endpoint, can we get the user's server?
func RefreshToken(refreshToken string) (*AuthResponse, error) {
	url := "https://bsky.social/xrpc/com.atproto.server.refreshSession"

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+refreshToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("reauth failed")
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

func GetUserInfo(token string, screen_name string) (*bridge.TwitterUser, error) {
	url := "https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile" + "?actor=" + screen_name

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch user info")
	}

	author := Author{}
	if err := json.NewDecoder(resp.Body).Decode(&author); err != nil {
		return nil, err
	}

	return AuthorTTB(author), nil
}

func GetUsersInfo(token string, items []string) ([]*bridge.TwitterUser, error) {
	url := "https://public.api.bsky.app/xrpc/app.bsky.actor.getProfiles" + "?actors=" + strings.Join(items, "&actors=")

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch user info")
	}

	var authors struct {
		Profiles []Author `json:"profiles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authors); err != nil {
		return nil, err
	}

	users := make([]*bridge.TwitterUser, len(authors.Profiles))
	for i, author := range authors.Profiles {
		users[i] = AuthorTTB(author)
	}

	return users, nil
}

func AuthorTTB(author Author) *bridge.TwitterUser {
	return &bridge.TwitterUser{
		ProfileSidebarFillColor:   "e0ff92",
		Name:                      author.DisplayName,
		ProfileSidebarBorderColor: "87bc44",
		ProfileBackgroundTile:     false,
		CreatedAt:                 author.CreatedAt,
		ProfileImageURL:           "http://10.0.0.77:3000/cdn/img/?url=" + url.QueryEscape(author.Avatar) + ":thumb",
		Location:                  "",
		ProfileLinkColor:          "0000ff",
		IsTranslator:              false,
		ContributorsEnabled:       false,
		URL:                       "",
		FavouritesCount:           0,
		UtcOffset:                 nil,
		ID:                        *bridge.BlueSkyToTwitterID(author.DID),
		ProfileUseBackgroundImage: false,
		ListedCount:               0,
		ProfileTextColor:          "000000",
		Protected:                 false,
		FollowersCount:            author.FollowersCount,
		Lang:                      "en",
		Notifications:             nil,
		Verified:                  false,
		ProfileBackgroundColor:    "c0deed",
		GeoEnabled:                false,
		Description:               author.Description,
		FriendsCount:              author.FollowsCount,
		StatusesCount:             author.PostsCount,
		ScreenName:                author.Handle,
	}
}

// https://docs.bsky.app/docs/api/app-bsky-feed-get-feed
func GetTimeline(token string, context string) (error, *Timeline) {
	url := "https://public.bsky.social/xrpc/app.bsky.feed.getTimeline"
	if context != "" {
		url = "https://public.bsky.social/xrpc/app.bsky.feed.getTimeline?context=" + context
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	// // Print the response body for debugging
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to fetch timeline"), nil
	}

	feeds := Timeline{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return err, nil
	}

	return nil, &feeds
}

func GetPost(token string, uri string, depth int, parentHeight int) (error, *ThreadRoot) {
	// Example URL at://did:plc:dqibjxtqfn6hydazpetzr2w4/app.bsky.feed.post/3lchbospvbc2j

	url := "https://public.bsky.social/xrpc/app.bsky.feed.getPostThread?depth=" + fmt.Sprintf("%d", depth) + "&parentHeight=" + fmt.Sprintf("%d", parentHeight) + "&uri=" + uri

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err, nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to fetch timeline"), nil
	}

	thread := ThreadRoot{}
	if err := json.NewDecoder(resp.Body).Decode(&thread); err != nil {
		return err, nil
	}

	return nil, &thread
}

func UpdateStatus(token string, status string) error {
	url := "https://public.bsky.social/xrpc/com.atproto.repo.createRecord"

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil
	}
	return errors.New("failed to update status")
}

func ReTweet(token string, id string, my_did string) (error, *ThreadRoot, *string) {
	url := "https://bsky.social/xrpc/com.atproto.repo.createRecord"

	err, thread := GetPost(token, id, 0, 1)

	if err != nil {
		return errors.New("failed to fetch post"), nil, nil
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err, nil, nil
	}
	payload := CreateRecordPayload{
		Collection: "app.bsky.feed.repost",
		Repo:       my_did,
		Record: RepostRecord{
			Type:      "app.bsky.feed.repost",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Subject: Subject{
				URI: thread.Thread.Post.URI,
				CID: thread.Thread.Post.CID,
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil, nil
	}

	req.Body = io.NopCloser(bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// if it works, we should get something like:
	// {"uri":"at://did:plc:khcyntihpu7snjszuojjgjc4/app.bsky.feed.repost/3lcm7b2pjio22","cid":"bafyreidw2uvnhns5bacdii7gozrou4rg25cpcxhe6cbhfws2c5hpsvycdm","commit":{"cid":"bafyreicu7db6k3vxbvtwiumggynbps7cuozsofbvo3kq7lz723smvpxne4","rev":"3lcm7b2ptb622"},"validationStatus":"valid"}
	resp, err := client.Do(req)
	if err != nil {
		return err, nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString), nil, nil
	}

	repost := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&repost); err != nil {
		return err, nil, nil
	}

	return nil, thread, &repost.URI
}

func LikePost(token string, id string, my_did string) (error, *ThreadRoot) {
	url := "https://bsky.social/xrpc/com.atproto.repo.createRecord"

	err, thread := GetPost(token, id, 0, 1)

	if err != nil {
		return errors.New("failed to fetch post"), nil
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err, nil
	}
	payload := CreateRecordPayload{
		Collection: "app.bsky.feed.like",
		Repo:       my_did,
		Record: RepostRecord{
			Type:      "app.bsky.feed.like",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Subject: Subject{
				URI: thread.Thread.Post.URI,
				CID: thread.Thread.Post.CID,
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil
	}

	req.Body = io.NopCloser(bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// if it works, we should get something like:
	// {"uri":"at://did:plc:khcyntihpu7snjszuojjgjc4/app.bsky.feed.repost/3lcm7b2pjio22","cid":"bafyreidw2uvnhns5bacdii7gozrou4rg25cpcxhe6cbhfws2c5hpsvycdm","commit":{"cid":"bafyreicu7db6k3vxbvtwiumggynbps7cuozsofbvo3kq7lz723smvpxne4","rev":"3lcm7b2ptb622"},"validationStatus":"valid"}
	resp, err := client.Do(req)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString), nil
	}

	likeRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&likeRes); err != nil {
		return err, nil
	}

	thread.Thread.Post.Viewer.Like = &strings.Split(likeRes.URI, "/app.bsky.feed.like/")[1]

	return nil, thread
}

func UnlikePost(token string, id string, my_did string) (error, *ThreadRoot) {
	url := "https://bsky.social/xrpc/com.atproto.repo.deleteRecord"

	err, thread := GetPost(token, id, 0, 1)

	if err != nil {
		return errors.New("failed to fetch post"), nil
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err, nil
	}
	payload := DeleteRecordPayload{
		Collection: "app.bsky.feed.like",
		Repo:       my_did,
		RKey:       strings.Split(*thread.Thread.Post.Viewer.Like, "/app.bsky.feed.like/")[1],
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload"), nil
	}

	req.Body = io.NopCloser(bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// if it works, we should get something like:
	// {"uri":"at://did:plc:khcyntihpu7snjszuojjgjc4/app.bsky.feed.repost/3lcm7b2pjio22","cid":"bafyreidw2uvnhns5bacdii7gozrou4rg25cpcxhe6cbhfws2c5hpsvycdm","commit":{"cid":"bafyreicu7db6k3vxbvtwiumggynbps7cuozsofbvo3kq7lz723smvpxne4","rev":"3lcm7b2ptb622"},"validationStatus":"valid"}
	resp, err := client.Do(req)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return errors.New("failed to retweet: " + bodyString), nil
	}

	likeRes := CreateRecordResult{}
	if err := json.NewDecoder(resp.Body).Decode(&likeRes); err != nil {
		return err, nil
	}

	thread.Thread.Post.Viewer.Like = &likeRes.URI // maybe?

	return nil, thread
}

func GetLikes(token string, uri string, limit int) (*Likes, error) {
	url := fmt.Sprintf("https://public.bsky.social/xrpc/app.bsky.feed.getLikes?limit=%d&uri=%s", limit, uri)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch timeline")
	}

	likes := Likes{}
	if err := json.NewDecoder(resp.Body).Decode(&likes); err != nil {
		return nil, err
	}

	return &likes, nil
}

func GetRetweetAuthors(token string, uri string, limit int) (*RepostedBy, error) {
	url := fmt.Sprintf("https://public.bsky.social/xrpc/app.bsky.feed.getRepostedBy?limit=%d&uri=%s", limit, uri)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// // Print the response body
	// bodyBytes, _ := io.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println("Response Body:", bodyString)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Println("Response Status:", resp.StatusCode)
		fmt.Println("Response Body:", bodyString)
		return nil, errors.New("failed to fetch timeline")
	}

	retweetAuthors := RepostedBy{}
	if err := json.NewDecoder(resp.Body).Decode(&retweetAuthors); err != nil {
		return nil, err
	}

	return &retweetAuthors, nil
}

package model

import "encoding/json"

type User struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	Nickname     string  `json:"nickname"`
	Avatar       *string `json:"avatar"`
	Bio          string  `json:"bio"`
	Role         string  `json:"role"`
	Status       string  `json:"status,omitempty"`
	CreatedAt    string  `json:"created_at"`
	PasswordHash string  `json:"-"`
}

type Author struct {
	ID       int64   `json:"id"`
	Username string  `json:"username"`
	Nickname string  `json:"nickname"`
	Avatar   *string `json:"avatar"`
}

type Exchange struct {
	ID            int64   `json:"id"`
	AuthorID      int64   `json:"-"`
	Kind          string  `json:"kind"`
	Title         string  `json:"title"`
	Content       string  `json:"content"`
	ImageURL      *string `json:"imageUrl"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	LikeCount     int     `json:"likeCount"`
	FavoriteCount int     `json:"favoriteCount"`
	CommentCount  int     `json:"commentCount"`
	Liked         bool    `json:"liked"`
	Favorited     bool    `json:"favorited"`
	Author        Author  `json:"author"`
}

type Comment struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	Author    Author `json:"author"`
}

type Mate struct {
	ID                  int64    `json:"id"`
	AuthorID            int64    `json:"-"`
	Title               string   `json:"title"`
	Tags                []string `json:"tags"`
	Details             string   `json:"details"`
	ImageURL            *string  `json:"imageUrl"`
	Location            *string  `json:"location"`
	Contact             *string  `json:"contact"`
	ContactHint         *string  `json:"contactHint"`
	Duration            string   `json:"duration"`
	Notes               *string  `json:"notes"`
	Status              string   `json:"status"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
	ApplicationCount    int      `json:"applicationCount"`
	MyApplicationStatus *string  `json:"myApplicationStatus"`
	IsOwner             bool     `json:"isOwner"`
	Author              Author   `json:"author"`
}

type Applicant struct {
	Author
	Bio string `json:"bio"`
}

type Application struct {
	ID             int64     `json:"id"`
	MateID         int64     `json:"-"`
	ApplicantID    int64     `json:"-"`
	OwnerID        int64     `json:"-"`
	Message        string    `json:"message"`
	Status         string    `json:"status"`
	CreatedAt      string    `json:"createdAt"`
	RespondedAt    *string   `json:"respondedAt"`
	ConversationID *int64    `json:"conversationId"`
	Applicant      Applicant `json:"applicant"`
}

type PostBrief struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type MyApplication struct {
	ID                int64     `json:"id"`
	Message           string    `json:"message"`
	Status            string    `json:"status"`
	CreatedAt         string    `json:"createdAt"`
	Post              PostBrief `json:"post"`
	ApplicantNickname string    `json:"applicantNickname"`
	OwnerNickname     string    `json:"ownerNickname"`
	ConversationID    *int64    `json:"conversationId"`
}

type OtherUser struct {
	Author
	Status string `json:"status"`
}

type Conversation struct {
	ID            int64     `json:"id"`
	UserAID       int64     `json:"-"`
	UserBID       int64     `json:"-"`
	OtherUser     OtherUser `json:"otherUser"`
	LastMessage   string    `json:"lastMessage"`
	LastMessageAt string    `json:"lastMessageAt"`
	UnreadCount   int       `json:"unreadCount"`
	PostTitle     string    `json:"postTitle"`
}

type Sender struct {
	ID       int64   `json:"id"`
	Nickname string  `json:"nickname"`
	Avatar   *string `json:"avatar"`
}

type Message struct {
	ID        int64   `json:"id"`
	Content   string  `json:"content"`
	CreatedAt string  `json:"createdAt"`
	ReadAt    *string `json:"readAt"`
	Sender    Sender  `json:"sender"`
}

type MessageContext struct {
	ID             int64  `json:"id"`
	Content        string `json:"content"`
	CreatedAt      string `json:"createdAt"`
	SenderID       int64  `json:"senderId"`
	SenderNickname string `json:"senderNickname"`
}

type Report struct {
	ID               int64            `json:"id"`
	TargetType       string           `json:"targetType"`
	TargetID         int64            `json:"targetId"`
	Reason           string           `json:"reason"`
	Details          string           `json:"details"`
	Snapshot         json.RawMessage  `json:"snapshot"`
	Status           string           `json:"status"`
	Action           *string          `json:"action"`
	AdminNote        *string          `json:"adminNote"`
	CreatedAt        string           `json:"createdAt"`
	ResolvedAt       *string          `json:"resolvedAt"`
	Reporter         Author           `json:"reporter"`
	ResolverNickname *string          `json:"resolverNickname"`
	MessageContext   []MessageContext `json:"messageContext"`
}

type AdminUser struct {
	User
	PostCount int `json:"postCount"`
}

type AdminContent struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Kind           string `json:"kind,omitempty"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
	AuthorNickname string `json:"authorNickname"`
	AuthorUsername string `json:"authorUsername"`
}

type Summary struct {
	Users               int `json:"users"`
	BannedUsers         int `json:"bannedUsers"`
	ExchangePosts       int `json:"exchangePosts"`
	MatePosts           int `json:"matePosts"`
	PendingReports      int `json:"pendingReports"`
	ActiveConversations int `json:"activeConversations"`
}

type Page[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type List[T any] struct {
	Items []T `json:"items"`
}

type ExchangeInput struct {
	Kind     string  `json:"kind"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	ImageURL *string `json:"imageUrl"`
}

type MateInput struct {
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Details  string   `json:"details"`
	ImageURL *string  `json:"imageUrl"`
	Location *string  `json:"location"`
	Contact  string   `json:"contact"`
	Duration string   `json:"duration"`
	Notes    *string  `json:"notes"`
}

type ReportInput struct {
	TargetType string `json:"targetType"`
	TargetID   int64  `json:"targetId"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

type ReportAction struct {
	Status    string `json:"status"`
	Action    string `json:"action"`
	AdminNote string `json:"adminNote"`
}

type Filter struct {
	Query    string
	Kind     string
	Tag      string
	Mine     bool
	Page     int
	PageSize int
}

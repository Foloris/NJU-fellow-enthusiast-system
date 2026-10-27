export type User = {
  id: number;
  username: string;
  email: string | null;
  nickname: string;
  avatar: string | null;
  bio: string;
  role: "user" | "admin";
  status?: "active" | "banned";
  created_at: string;
};

export type RegistrationConfig = {
  emailDomain: string;
  codeLifetimeSeconds: number;
  resendAfterSeconds: number;
  emailDeliveryAvailable: boolean;
};

export type EmailCodeResult = {
  message: string;
  expiresIn: number;
  retryAfter: number;
};

export type Author = {
  id: number;
  username: string;
  nickname: string;
  avatar: string | null;
};

export type ExchangePost = {
  id: number;
  kind: "share" | "news";
  title: string;
  content: string;
  imageUrl: string | null;
  status: "published" | "removed";
  createdAt: string;
  updatedAt: string;
  likeCount: number;
  favoriteCount: number;
  commentCount: number;
  liked: boolean;
  favorited: boolean;
  author: Author;
};

export type Comment = {
  id: number;
  content: string;
  createdAt: string;
  author: Author;
};

export type MatePost = {
  id: number;
  title: string;
  tags: string[];
  details: string;
  imageUrl: string | null;
  location: string | null;
  contact: string | null;
  contactHint: string | null;
  duration: string;
  notes: string | null;
  status: "published" | "removed";
  createdAt: string;
  updatedAt: string;
  applicationCount: number;
  myApplicationStatus: "pending" | "accepted" | "rejected" | null;
  isOwner: boolean;
  author: Author;
};

export type Application = {
  id: number;
  message: string;
  status: "pending" | "accepted" | "rejected";
  createdAt: string;
  respondedAt: string | null;
  conversationId: number | null;
  applicant: Author & { bio: string };
};

export type Conversation = {
  id: number;
  otherUser: Author & { status: string };
  lastMessage: string;
  lastMessageAt: string;
  unreadCount: number;
  postTitle: string;
};

export type ChatMessage = {
  id: number;
  content: string;
  createdAt: string;
  readAt: string | null;
  sender: Pick<Author, "id" | "nickname" | "avatar">;
};

export type Paginated<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};


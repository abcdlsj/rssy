export interface Feed {
  id: string;
  title: string;
  author?: string;
  url: string;
  groupId: string;
  lastUpdated?: string;
}

export interface FeedGroup {
  id: string;
  name: string;
}

export interface Article {
  id: string;
  feedId: string;
  title: string;
  link: string;
  content: string;
  pubDate: string;
  isRead: boolean;
  isBookmarked: boolean;
}
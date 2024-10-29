import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Feed, FeedGroup, Article } from '../types';
import { fetchRssFeed } from '../lib/rss2feed';

type FeedStore = {
  feeds: Feed[];
  groups: FeedGroup[];
  articles: Article[];
  selectedFeed: string | null;
  selectedGroup: string | null;
  selectedArticleId: string | null;
  isLoading: boolean;
  error: string | null;
  showBookmarks: boolean;

  addFeed: (feed: Omit<Feed, 'id'>) => Promise<void>;
  removeFeed: (id: string) => void;
  addGroup: (name: string) => void;
  removeGroup: (id: string) => void;
  selectFeed: (id: string | null) => void;
  selectGroup: (id: string | null) => void;
  refreshFeed: (feedId: string) => Promise<void>;
  selectAndToggleReadStatus: (articleId: string) => void;
  toggleBookmark: (articleId: string) => void;
  toggleShowBookmarks: () => void;
};

export const useFeedStore = create<FeedStore>()(
  persist(
    (set, get) => ({
      feeds: [],
      groups: [],
      articles: [],
      selectedFeed: null,
      selectedGroup: null,
      isLoading: false,
      error: null,
      selectedArticleId: null,
      showBookmarks: false,

      addFeed: async (feedData) => {
        set({ isLoading: true, error: null });
        try {
          const feed = {
            ...feedData,
            id: crypto.randomUUID(),
            lastUpdated: new Date().toISOString(),
          };

          const feedContent = await fetchRssFeed(feed.url);
          feed.title = feedContent.title || feed.url;
          feed.author = feedContent.author || '';

          const newArticles = feedContent.items.map((item: {
            title?: string;
            link?: string;
            content?: string;
            pubDate?: string;
            guid?: string;
          }) => ({
            id: item.guid || crypto.randomUUID(),
            feedId: feed.id,
            title: item.title || 'Untitled',
            link: item.link || '',
            content: item.content || '',
            pubDate: item.pubDate || new Date().toISOString(),
            isRead: false,
            isBookmarked: false,
          }));

          set((state) => ({
            feeds: [...state.feeds, feed],
            articles: [...state.articles, ...newArticles],
            isLoading: false,
          }));
        } catch (error) {
          console.error('Failed to add feed:', error);
          set({ error: 'Failed to add feed', isLoading: false });
        }
      },

      removeFeed: (id) => {
        set((state) => ({
          feeds: state.feeds.filter((f) => f.id !== id),
          articles: state.articles.filter((a) => a.feedId !== id),
          selectedFeed: state.selectedFeed === id ? null : state.selectedFeed,
        }));
      },

      addGroup: (name) => {
        set((state) => ({
          groups: [...state.groups, { id: crypto.randomUUID(), name }],
        }));
      },

      removeGroup: (id) => {
        set((state) => ({
          groups: state.groups.filter((g) => g.id !== id),
          feeds: state.feeds.map((f) =>
            f.groupId === id ? { ...f, groupId: '' } : f
          ),
        }));
      },

      selectFeed: (id) => {
        set({ selectedFeed: id, selectedGroup: null });
      },

      selectGroup: (id) => {
        set({ selectedGroup: id, selectedFeed: null });
      },

      refreshFeed: async (feedId) => {
        set({ isLoading: true, error: null });
        try {
          const { feeds, articles } = get();
          const feed = feeds.find((f) => f.id === feedId);

          if (!feed) throw new Error('Feed not found');

          const feedContent = await fetchRssFeed(feed.url);
          const existingArticles = articles.filter(a => a.feedId === feedId);

          const newArticles = feedContent.items
            .filter((item: { guid?: string; link?: string }) => {
              const identifier = item.guid || item.link;
              return !existingArticles.some(existing => 
                (existing.id === identifier) || (existing.link === identifier)
              );
            })
            .map((item: {
              title?: string;
              link?: string;
              content?: string;
              pubDate?: string;
              guid?: string;
            }) => ({
              id: item.guid || crypto.randomUUID(),
              feedId,
              title: item.title || 'Untitled',
              link: item.link || '',
              content: item.content || '',
              pubDate: item.pubDate || new Date().toISOString(),
              isRead: false,
              isBookmarked: false,
            }));

          set((state) => ({
            articles: [
              ...state.articles.filter((a) => a.feedId !== feedId),
              ...existingArticles,
              ...newArticles,
            ],
            feeds: state.feeds.map((f) =>
              f.id === feedId
                ? { ...f, lastUpdated: new Date().toISOString() }
                : f
            ),
            isLoading: false,
          }));
        } catch (error) {
          console.error('Failed to refresh feed:', error);
          set({ error: 'Failed to refresh feed', isLoading: false });
        }
      },

      selectAndToggleReadStatus: (articleId) => {
        set((state) => ({
          articles: state.articles.map((article) =>
            article.id === articleId
              ? { ...article, isRead: true }
              : article
          ),
          selectedArticleId: articleId,
        }));
      },

      toggleBookmark: (articleId) => {
        set((state) => ({
          articles: state.articles.map((article) =>
            article.id === articleId
              ? { ...article, isBookmarked: !article.isBookmarked }
              : article
          ),
        }));
      },

      toggleShowBookmarks: () => 
        set((state) => ({
          showBookmarks: !state.showBookmarks,
          selectedFeed: null,
          selectedGroup: null,
        })),
    }),
    {
      name: 'rss-feed-storage',
    }
  )
);
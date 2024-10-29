import { format } from 'date-fns';
import { Bookmark, BookmarkCheck, Circle } from 'lucide-react';
import { useFeedStore } from '../stores/feedStore';

export function ArticleList() {
  const { 
    articles, 
    selectedFeed, 
    selectedGroup,
    showBookmarks,
    feeds,
    toggleBookmark,
    selectAndToggleReadStatus
  } = useFeedStore();

  const filteredArticles = articles.filter(article => {
    if (showBookmarks) return article.isBookmarked;
    if (selectedFeed) return article.feedId === selectedFeed;
    if (selectedGroup) {
      const groupFeeds = feeds.filter(f => f.groupId === selectedGroup);
      return groupFeeds.some(f => f.id === article.feedId);
    }
    return true;
  });

  return (
    <div className="w-96 h-screen border-r border-gray-200 overflow-y-auto">
      {filteredArticles.map(article => (
        <article
          key={article.id}
          className={`p-4 border-b border-gray-200 cursor-pointer hover:bg-gray-50 ${
            article.isRead ? 'opacity-60' : ''
          }`}
          onClick={() => selectAndToggleReadStatus(article.id)}
        >
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-2 flex-1">
              {!article.isRead && (
                <Circle className="w-2 h-2 mt-1.5 flex-shrink-0 fill-blue-600 text-blue-600" />
              )}
              <h3 className="text-sm font-medium text-gray-900">
                {article.title}
              </h3>
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                toggleBookmark(article.id);
              }}
              className="flex-shrink-0"
            >
              {article.isBookmarked ? (
                <BookmarkCheck className="w-5 h-5 text-blue-600" />
              ) : (
                <Bookmark className="w-5 h-5 text-gray-400" />
              )}
            </button>
          </div>
          <time className="text-xs text-gray-500 mt-1 block">
            {format(new Date(article.pubDate), 'MMM d, yyyy')}
          </time>
        </article>
      ))}
    </div>
  );
}
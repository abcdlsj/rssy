import { Share2, Bookmark, BookmarkCheck, ExternalLink } from 'lucide-react';
import { useFeedStore } from '../stores/feedStore';
import { format } from 'date-fns';

export function ArticleViewer() {
  const { articles, selectedArticleId, toggleBookmark } = useFeedStore();
  const selectedArticle = articles.find(a => a.id === selectedArticleId);

  if (!selectedArticle) {
    return (
      <div className="flex-1 h-screen flex items-center justify-center text-gray-500">
        <p>Select an article to read</p>
      </div>
    );
  }

  return (
    <div className="flex-1 h-screen overflow-y-auto bg-white">
      <div className="max-w-3xl mx-auto">
        <header className="sticky top-0 bg-white/80 backdrop-blur-sm border-b border-gray-200 px-6 py-4">
          <div className="flex items-center justify-between mb-2">
            <time className="text-sm text-gray-500">
              {format(new Date(selectedArticle.pubDate), 'MMMM d, yyyy')}
            </time>
            <div className="flex items-center gap-2">
              <button
                onClick={() => toggleBookmark(selectedArticle.id)}
                className="p-2 hover:bg-gray-100 rounded-full"
                title={selectedArticle.isBookmarked ? "Remove Bookmark" : "Add Bookmark"}
              >
                {selectedArticle.isBookmarked ? (
                  <BookmarkCheck className="w-5 h-5 text-blue-600" />
                ) : (
                  <Bookmark className="w-5 h-5 text-gray-400" />
                )}
              </button>
              <a
                href={selectedArticle.link}
                target="_blank"
                rel="noopener noreferrer"
                className="p-2 hover:bg-gray-100 rounded-full"
                title="Open Original"
              >
                <ExternalLink className="w-5 h-5 text-gray-400" />
              </a>
              <button
                onClick={() => {
                  navigator.share?.({
                    title: selectedArticle.title,
                    url: selectedArticle.link
                  }).catch(() => {
                    navigator.clipboard.writeText(selectedArticle.link);
                  });
                }}
                className="p-2 hover:bg-gray-100 rounded-full"
                title="Share Article"
              >
                <Share2 className="w-5 h-5 text-gray-400" />
              </button>
            </div>
          </div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            {selectedArticle.title}
          </h1>
        </header>

        <article className="px-6 py-8">
          <div 
            className="prose prose-base max-w-none
              prose-p:text-gray-600 prose-p:leading-relaxed
              prose-headings:font-semibold
              prose-headings:text-gray-900
              prose-h1:text-2xl
              prose-h2:text-xl
              prose-h3:text-lg
              prose-a:text-blue-600 hover:prose-a:text-blue-500
              prose-img:rounded-lg prose-img:shadow-md
              prose-pre:bg-gray-50 prose-pre:border prose-pre:border-gray-200
              prose-code:text-gray-800 prose-pre:p-4
              [&_pre]:overflow-x-auto [&_pre_code]:text-sm
              [&_pre_code]:text-gray-800 [&_pre_code]:bg-transparent"
            dangerouslySetInnerHTML={{ __html: selectedArticle.content }} 
          />
        </article>
      </div>
    </div>
  );
}
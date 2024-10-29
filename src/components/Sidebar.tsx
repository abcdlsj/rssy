import { FolderPlus, Plus, Trash2, Bookmark } from 'lucide-react';
import { useFeedStore } from '../stores/feedStore';
import { Feed } from '../types';

export function Sidebar() {
  const { 
    feeds, 
    groups, 
    selectedFeed, 
    selectedGroup,
    addGroup,
    removeGroup,
    selectFeed,
    selectGroup,
    removeFeed,
    showBookmarks,
    toggleShowBookmarks
  } = useFeedStore();

  const ungroupedFeeds = feeds.filter(feed => !feed.groupId);

  const handleAddGroup = () => {
    const name = prompt('Enter group name:');
    if (name) addGroup(name);
  };

  const FeedItem = ({ feed }: { feed: Feed }) => (
    <div className="flex items-center gap-2 w-full mb-1">
      <button
        onClick={() => selectFeed(feed.id)}
        className={`flex-1 text-left px-4 py-2 rounded-lg ${
          selectedFeed === feed.id
            ? 'bg-blue-100 text-blue-700'
            : 'hover:bg-gray-100'
        }`}
      >
        {feed.title}
      </button>
      <button
        onClick={() => removeFeed(feed.id)}
        className="p-1 hover:bg-gray-200 rounded-full"
        title="Remove Feed"
      >
        <Trash2 className="w-4 h-4" />
      </button>
    </div>
  );

  return (
    <aside className="w-64 h-screen bg-gray-50 border-r border-gray-200 p-4 overflow-y-auto">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold">Feeds</h2>
        <div className="flex gap-2">
          <button
            onClick={() => toggleShowBookmarks()}
            className={`p-2 rounded-full ${
              showBookmarks 
                ? 'bg-blue-100 text-blue-600' 
                : 'hover:bg-gray-200 text-gray-600'
            }`}
            title="Show Bookmarks"
          >
            <Bookmark className="w-5 h-5" />
          </button>
          <button
            onClick={handleAddGroup}
            className="p-2 hover:bg-gray-200 rounded-full"
            title="Add Group"
          >
            <FolderPlus className="w-5 h-5" />
          </button>
        </div>
      </div>

      {ungroupedFeeds.length > 0 && (
        <div className="mb-4">
          <div className="flex items-center mb-2">
            <span className="text-sm font-medium text-gray-700">Ungrouped</span>
          </div>
          {ungroupedFeeds.map(feed => (
            <FeedItem key={feed.id} feed={feed} />
          ))}
        </div>
      )}

      {groups.map(group => (
        <div key={group.id} className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <button
              onClick={() => selectGroup(group.id)}
              className={`text-sm font-medium ${
                selectedGroup === group.id ? 'text-blue-600' : 'text-gray-700'
              }`}
            >
              {group.name}
            </button>
            <button
              onClick={() => removeGroup(group.id)}
              className="p-1 hover:bg-gray-200 rounded-full"
              title="Remove Group"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
          
          {feeds
            .filter(feed => feed.groupId === group.id)
            .map(feed => (
              <FeedItem key={feed.id} feed={feed} />
            ))}
        </div>
      ))}

      <div className="mt-4">
        <button
          onClick={() => {
            const url = prompt('Enter RSS feed URL:');
            if (url) {
              useFeedStore.getState().addFeed({
                title: url,
                url,
                groupId: selectedGroup || ''
              });
            }
          }}
          className="flex items-center gap-2 px-4 py-2 text-sm text-blue-600 hover:bg-blue-50 rounded-lg w-full"
        >
          <Plus className="w-4 h-4" />
          Add Feed
        </button>
      </div>
    </aside>
  );
}
import { Sidebar } from './components/Sidebar';
import { ArticleList } from './components/ArticleList';
import { ArticleViewer } from './components/ArticleViewer';
import { useFeedStore } from './stores/feedStore';

function App() {
  const { isLoading, error } = useFeedStore();

  return (
    <div className="flex h-screen bg-white">
      <Sidebar />
      <ArticleList />
      <ArticleViewer />
      
      {isLoading && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center">
          <div className="bg-white p-4 rounded-lg shadow-lg">
            Loading...
          </div>
        </div>
      )}
      
      {error && (
        <div className="fixed bottom-4 right-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}
    </div>
  );
}

export default App;
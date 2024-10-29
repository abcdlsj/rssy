import { getArticleContent } from './jina_reader';

export async function fetchRssFeed(url: string) {
    try {
      const response = await fetch(`https://api.rss2json.com/v1/api.json?rss_url=${encodeURIComponent(url)}`);
      if (!response.ok) throw new Error('Failed to fetch RSS feed');
      const data = await response.json();
      
      if (data.status !== 'ok') throw new Error(data.message || 'Failed to parse RSS feed');
      
      const items = await Promise.all(
        data.items.map(async (item: {
          title?: string;
          link?: string;
          content?: string;
          pubDate?: string;
          description?: string;
          guid?: string;
        }) => {
          let content = item.content;
          if ((!content || content === item.description) && item.link) {
            content = await getArticleContent(item.link);
          }
          return {
            title: item.title,
            link: item.link,
            content: content || '',
            pubDate: item.pubDate,
            guid: item.guid,
          };
        })
      );
  
      return {
        title: data.feed.title,
        author: data.feed.author,
        items,
      };
    } catch (error) {
      console.error('Error fetching RSS feed:', error);
      throw error;
    }
  }
  
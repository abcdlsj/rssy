import { marked } from 'marked';

const JINA_BASE_URL = 'https://r.jina.ai';


function removeJinaMeta(rawContent: string): string {
  const lines = rawContent.split('\n');
  let contentStartIndex = 0;
  let foundMarkdownContent = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    
    if (line === 'Markdown Content:') {
      contentStartIndex = i + 1;
      foundMarkdownContent = true;
      break;
    }
    
    if (!foundMarkdownContent && i > 3) {
      const isMetadata = 
        line.startsWith('Title:') ||
        line.startsWith('URL Source:') ||
        line.startsWith('=') ||
        line === '' ||
        /^\d{4}-\d{2}-\d{2}$/.test(line);
      
      if (!isMetadata) {
        contentStartIndex = i;
        break;
      }
    }
  }

  const content = lines.slice(contentStartIndex).join('\n')
    .replace(/^={2,}\s*$\n/gm, '')
    .replace(/^#{1,6}\s+.*$\n/gm, '')
    .trim();

  return content;
}

async function convertMarkdownToHtml(markdown: string): Promise<string> {
  marked.setOptions({
    gfm: true,
    breaks: true,
  });

  const html = marked(markdown);
  return html;
}

export async function getArticleContent(url: string): Promise<string> {
  try {
    const response = await fetch(`${JINA_BASE_URL}/${encodeURIComponent(url)}`);
    if (!response.ok) {
      throw new Error('Failed to fetch content');
    }

    const rawContent = await response.text();
    const content = removeJinaMeta(rawContent);

    return await convertMarkdownToHtml(content);
  } catch (error) {
    console.error('Error fetching article content:', error);
    return '';
  }
}

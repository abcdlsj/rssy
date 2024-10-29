/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      typography: {
        DEFAULT: {
          css: {
            maxWidth: 'none',
            pre: {
              backgroundColor: '#f8f9fa',
              color: '#334155',
              padding: '1rem',
              borderRadius: '0.5rem',
              margin: '1.5rem 0',
            },
            'pre code': {
              backgroundColor: 'transparent',
              borderWidth: '0',
              borderRadius: '0',
              padding: '0',
              fontSize: '0.875rem',
              lineHeight: '1.5',
              fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
            },
            code: {
              color: '#334155',
              '&::before': {
                content: '""',
              },
              '&::after': {
                content: '""',
              },
            },
            fontSize: '0.9375rem',
            lineHeight: '1.6',
            p: {
              fontSize: '0.9375rem',
              lineHeight: '1.6',
              marginTop: '1em',
              marginBottom: '1em',
            },
            h1: {
              fontSize: '1.5rem',
            },
            h2: {
              fontSize: '1.25rem',
            },
            h3: {
              fontSize: '1.125rem',
            },
          },
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography')
  ],
};
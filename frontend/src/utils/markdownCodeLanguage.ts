const languageAliases: Readonly<Record<string, string>> = {
  'c++': 'cpp',
  'c#': 'csharp',
  cs: 'csharp',
  dockerfile: 'docker',
  golang: 'go',
  htm: 'markup',
  html: 'markup',
  js: 'javascript',
  kt: 'kotlin',
  kts: 'kotlin',
  md: 'markdown',
  mermaid: 'mermaid',
  mmd: 'mermaid',
  ps1: 'powershell',
  pwsh: 'powershell',
  py: 'python',
  rs: 'rust',
  sh: 'bash',
  shell: 'bash',
  ts: 'typescript',
  xml: 'markup',
  yml: 'yaml',
};

export const resolveMarkdownCodeLanguage = (className?: string) => {
  const match = /language-([^\s]+)/i.exec(className || '');
  if (!match) {
    return '';
  }

  const language = match[1].toLowerCase();
  return languageAliases[language] || language;
};

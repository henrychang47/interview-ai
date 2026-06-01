import type { ReactNode } from 'react'

type MarkdownTextProps = {
  text: string
  className?: string
}

type MarkdownBlock =
  | { type: 'heading'; level: number; text: string }
  | { type: 'list'; ordered: boolean; items: string[] }
  | { type: 'paragraph'; lines: string[] }

const headingClasses = 'font-headline text-body-lg font-bold leading-7 text-on-surface'
const paragraphClasses = 'text-body-md leading-7 text-on-surface'
const listClasses = 'space-y-xs pl-lg text-body-md leading-7 text-on-surface'
const inlineCodeClasses =
  'rounded bg-surface-container-high px-xs py-[1px] font-mono text-[0.92em] text-on-surface'

export function MarkdownText({ text, className = '' }: MarkdownTextProps) {
  const blocks = parseMarkdownBlocks(text)

  return (
    <div className={`space-y-sm ${className}`}>
      {blocks.map((block, index) => renderBlock(block, index))}
    </div>
  )
}

function parseMarkdownBlocks(text: string): MarkdownBlock[] {
  const lines = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n')
  const blocks: MarkdownBlock[] = []
  let index = 0

  while (index < lines.length) {
    const line = lines[index]
    if (line.trim() === '') {
      index += 1
      continue
    }

    const headingMatch = line.match(/^(#{1,6})\s+(.+)$/)
    if (headingMatch) {
      blocks.push({
        type: 'heading',
        level: headingMatch[1].length,
        text: headingMatch[2].trim(),
      })
      index += 1
      continue
    }

    const unorderedMatch = line.match(/^\s*[-*]\s+(.+)$/)
    const orderedMatch = line.match(/^\s*\d+[.)]\s+(.+)$/)
    if (unorderedMatch || orderedMatch) {
      const ordered = Boolean(orderedMatch)
      const items: string[] = []

      while (index < lines.length) {
        const itemMatch = ordered
          ? lines[index].match(/^\s*\d+[.)]\s+(.+)$/)
          : lines[index].match(/^\s*[-*]\s+(.+)$/)
        if (!itemMatch) {
          break
        }
        items.push(itemMatch[1].trim())
        index += 1
      }

      blocks.push({ type: 'list', ordered, items })
      continue
    }

    const paragraphLines: string[] = []
    while (index < lines.length) {
      const currentLine = lines[index]
      if (
        currentLine.trim() === '' ||
        /^(#{1,6})\s+(.+)$/.test(currentLine) ||
        /^\s*[-*]\s+(.+)$/.test(currentLine) ||
        /^\s*\d+[.)]\s+(.+)$/.test(currentLine)
      ) {
        break
      }
      paragraphLines.push(currentLine)
      index += 1
    }

    blocks.push({ type: 'paragraph', lines: paragraphLines })
  }

  return blocks
}

function renderBlock(block: MarkdownBlock, index: number) {
  if (block.type === 'heading') {
    const HeadingTag = block.level <= 2 ? 'h4' : block.level === 3 ? 'h5' : 'h6'
    return (
      <HeadingTag key={index} className={headingClasses}>
        {renderInlineMarkdown(block.text)}
      </HeadingTag>
    )
  }

  if (block.type === 'list') {
    const ListTag = block.ordered ? 'ol' : 'ul'
    return (
      <ListTag
        key={index}
        className={`${listClasses} ${block.ordered ? 'list-decimal' : 'list-disc'}`}
      >
        {block.items.map((item, itemIndex) => (
          <li key={`${index}-${itemIndex}`}>{renderInlineMarkdown(item)}</li>
        ))}
      </ListTag>
    )
  }

  return (
    <p key={index} className={paragraphClasses}>
      {block.lines.map((line, lineIndex) => (
        <span key={`${index}-${lineIndex}`}>
          {lineIndex > 0 ? <br /> : null}
          {renderInlineMarkdown(line)}
        </span>
      ))}
    </p>
  )
}

function renderInlineMarkdown(text: string): ReactNode[] {
  const nodes: ReactNode[] = []
  const pattern = /(\*\*[^*]+\*\*|`[^`]+`)/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > lastIndex) {
      nodes.push(text.slice(lastIndex, match.index))
    }

    const token = match[0]
    if (token.startsWith('**')) {
      nodes.push(<strong key={nodes.length}>{token.slice(2, -2)}</strong>)
    } else {
      nodes.push(
        <code key={nodes.length} className={inlineCodeClasses}>
          {token.slice(1, -1)}
        </code>,
      )
    }
    lastIndex = match.index + token.length
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex))
  }

  return nodes
}

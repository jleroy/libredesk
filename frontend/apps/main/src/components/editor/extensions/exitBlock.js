import { TextSelection } from '@tiptap/pm/state'

// Enter on the empty last line of a container escapes it, the way a list does.
// Without this the only way out of a callout or collapsible is the mouse.
export function exitOnEmptyTrailingLine(editor, ancestorName) {
  const { state, view } = editor
  const { $from, empty } = state.selection
  if (!empty) return false
  if ($from.parent.type.name !== 'paragraph' || $from.parent.content.size !== 0) return false

  let depth = $from.depth - 1
  while (depth > 0 && $from.node(depth).type.name !== ancestorName) depth--
  if (depth <= 0) return false

  // Only the last line escapes; anywhere else Enter still adds a line inside.
  if ($from.after(depth) - $from.after() !== $from.depth - depth) return false

  const emptyLineSize = $from.after() - $from.before()
  const insertAt = $from.after(depth) - emptyLineSize
  const tr = state.tr.delete($from.before(), $from.after())
  tr.insert(insertAt, state.schema.nodes.paragraph.create())
  tr.setSelection(TextSelection.near(tr.doc.resolve(insertAt + 1)))
  view.dispatch(tr.scrollIntoView())
  return true
}

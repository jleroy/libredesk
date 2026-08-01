import { Node, mergeAttributes } from '@tiptap/vue-3'
import { TextSelection } from '@tiptap/pm/state'

// Expandable section rendered as native <details>/<summary>. Editor CSS force-shows
// the collapsed body to keep it editable.
export const Details = Node.create({
  name: 'details',
  group: 'block',
  content: 'detailsSummary detailsContent',
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: 'details' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['details', mergeAttributes(HTMLAttributes, { class: 'hc-details' }), 0]
  },

  addCommands() {
    return {
      setDetails:
        () =>
        ({ chain }) =>
          chain()
            .insertContent({
              type: this.name,
              content: [
                { type: 'detailsSummary', content: [{ type: 'text', text: 'Summary' }] },
                { type: 'detailsContent', content: [{ type: 'paragraph' }] }
              ]
            })
            .run()
    }
  }
})

export const DetailsSummary = Node.create({
  name: 'detailsSummary',
  content: 'inline*',
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: 'summary' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['summary', mergeAttributes(HTMLAttributes, { class: 'hc-details-summary' }), 0]
  },

  // The summary is isolating, so a plain Enter would be swallowed; jump into the body instead.
  addKeyboardShortcuts() {
    return {
      Enter: () => {
        const { state, view } = this.editor
        const { $from, empty } = state.selection
        if (!empty || $from.parent.type.name !== this.name) return false
        const $body = state.doc.resolve($from.after() + 1)
        view.dispatch(state.tr.setSelection(TextSelection.near($body, 1)).scrollIntoView())
        return true
      }
    }
  }
})

export const DetailsContent = Node.create({
  name: 'detailsContent',
  content: 'block+',
  defining: true,
  isolating: true,

  parseHTML() {
    return [{ tag: 'div.hc-details-content' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { class: 'hc-details-content' }), 0]
  }
})

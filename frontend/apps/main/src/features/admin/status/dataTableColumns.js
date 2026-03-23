import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'
import { CONVERSATION_DEFAULT_STATUSES_LIST } from '@/constants/conversation.js'

export const createColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      const isDefault = CONVERSATION_DEFAULT_STATUSES_LIST.includes(row.getValue('name'))
      return h('div', { class: 'text-center' },
        onEdit && !isDefault
          ? h('span', {
              class: 'text-primary hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, row.getValue('name'))
          : row.getValue('name')
      )
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('created_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const status = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          status
        })
      )
    }
  }
]

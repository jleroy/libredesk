import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'

export const columns = [

    {
        accessorKey: 'name',
        header: function () {
            return h('div', { class: 'text-center' }, 'Provider')
        },
        cell: function ({ row }) {
            return h('div', { class: 'text-center font-medium' }, row.getValue('name'))
        }
    },
    {
        accessorKey: 'updated_at',
        header: function () {
            return h('div', { class: 'text-center' }, 'Modified at')
        },
        cell: function ({ row }) {
            return h('div', { class: 'text-center' }, format(row.getValue('updated_at'), 'PPpp'))
        }
    },
    {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
            const role = row.original
            return h(
                'div',
                { class: 'relative' },
                h(dropdown, {
                    role
                })
            )
        }
    }
]

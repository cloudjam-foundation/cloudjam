<script lang="ts">
	import { getContext } from 'svelte';
	import * as DropdownMenu from '$lib/components/shad/ui/dropdown-menu';
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	type Props = {
		onclick?: (() => Promise<void>) | (() => void);
		disabled?: boolean;
		children: Snippet;
		class?: ClassValue;
		[key: string]: unknown;
	};

	let { onclick, disabled, children, class: className, ...restProps }: Props = $props();

	const menu = getContext<{ close: () => void } | undefined>('dataTableAction');
</script>

<DropdownMenu.Item
	{...restProps}
	class={className}
	{disabled}
	onclick={async (e) => {
		// Stay open while the action runs (e.g. to show a spinner), then close. Without
		// closing, items that open a dialogue left the menu "open" behind it, so the
		// next click on the trigger only closed it again.
		e.preventDefault();
		try {
			if (onclick) await onclick();
		} finally {
			menu?.close();
		}
	}}
>
	{@render children()}
</DropdownMenu.Item>

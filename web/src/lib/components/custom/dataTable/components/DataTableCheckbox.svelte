<script lang="ts">
	import type { ComponentProps } from 'svelte';
	import { Checkbox } from '$lib/components/shad/ui/checkbox';

	let {
		checked = false,
		onCheckedChange = (v) => (checked = v),
		...restProps
	}: ComponentProps<typeof Checkbox> = $props();
</script>

<Checkbox data-rybbit-event="table-selectRow" {...restProps} bind:checked={() => checked, onCheckedChange} />

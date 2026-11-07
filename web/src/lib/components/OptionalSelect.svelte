<!-- Opinionated selector that can dynamically load data and also works if no suggestion got loaded -->
<script lang="ts">
	import Input from '$lib/components/Input.svelte';
	import Select from '$lib/components/Select.svelte';

	export type Suggestion = { id: string; title: string };

	let {
		value = $bindable(),
		suggestions,
		placeholder,
		onselect,
		...restProps
	}: {
		value: string;
		suggestions: Suggestion[];
		placeholder: string;
		onselect?: () => void;
		class?: string;
	} = $props();
</script>

{#if suggestions.length < 1}
	<Input class="max-w-96" bind:value {placeholder} {...restProps} />
{:else}
	<Select
		bind:value
		{placeholder}
		items={suggestions.map((suggestion) => ({ value: suggestion.id, label: suggestion.title }))}
		onchange={() => onselect?.()}
		class="w-72"
		{...restProps}
	/>
{/if}

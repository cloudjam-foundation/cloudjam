<script lang="ts">
	import { getSmartForm } from '$lib/components/forms/form.svelte';
	import { onMount } from 'svelte';
	import Input from '$lib/components/inputs/Input.svelte';
	import type { HTMLInputTypeAttribute } from 'svelte/elements';

	let {
		id,
		label,
		validator,
		placeholder,
		type = 'text',
		value = $bindable(),
		checked = $bindable(),
		files = $bindable(),
		class: className
	}: {
		id: string;
		label?: string;
		validator?: (value: string | number | boolean | undefined) => string | undefined;
		placeholder?: string;
		type: HTMLInputTypeAttribute;
		value?: string | number | boolean;
		checked?: boolean;
		files?: FileList;
		class?: string;
	} = $props();

	let smartForm = getSmartForm();

	let current = $derived(type === 'checkbox' ? checked : value);
	let error = $derived((smartForm?.live ?? true) ? validator?.(current) : undefined);

	onMount(() =>
		smartForm?.register({
			validate: () => validator?.(current),
			focus: () => document.getElementById(id)?.focus()
		})
	);
</script>

<Input
	{id}
	{type}
	{label}
	{placeholder}
	bind:value
	bind:files
	bind:checked
	validation={error}
	forceValidation
	aria-invalid={error ? true : undefined}
	aria-describedby={error ? `${id}-error` : undefined}
	class={className}
/>

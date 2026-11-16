<script lang="ts">
	import type { Snippet } from 'svelte';
	import { type FormMethods, registerSmartForm, SmartFormClass } from '$lib/components/forms/form.svelte';
	import type { SubmitState } from '$lib';

	let {
		form = $bindable(),
		submitState = $bindable(),
		action,
		method = 'GET',
		class: className,
		onsubmit,
		children
	}: {
		form?: HTMLFormElement;
		submitState: SubmitState;
		action?: string | null | undefined;
		method?: FormMethods;
		class?: string;
		onsubmit: () => void | Promise<void>;
		children: Snippet;
	} = $props();

	let smartForm = registerSmartForm(new SmartFormClass());

	async function _submit(e: SubmitEvent) {
		e.preventDefault();
		submitState.loading = true;
		if (!smartForm.validate()) {
			smartForm.live = true;
			submitState.loading = false;
			return;
		}

		smartForm.live = false;
		await onsubmit();
		submitState.loading = false;
	}
</script>

<form bind:this={form} {action} {method} class={className} novalidate onsubmit={_submit}>
	{@render children()}
</form>

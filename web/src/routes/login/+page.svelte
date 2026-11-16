<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, setToken, Submit, type SubmitState } from '$lib';
	import logo from '$lib/assets/favicon.svg';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Button from '$lib/components/buttons/Button.svelte';
	import Input from '$lib/components/inputs/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { LoginRequestSchema } from '$lib/sdk/v1/auth/auth_pb';
	import { create } from '@bufbuild/protobuf';
	import { resolve } from '$app/paths';
	import { loadActive } from '$lib/auth.svelte';
	import SmartForm from '$lib/components/forms/SmartForm.svelte';
	import SmartField from '$lib/components/forms/SmartField.svelte';

	let loginState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let loginRequest = $state(create(LoginRequestSchema, {}));
</script>

<svelte:head>
	<title>Login | CloudJam</title>
	<meta property="og:title" content="Login | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full items-center justify-center">
	<Card class="mt-[10%] w-96">
		<SmartForm
			bind:submitState={loginState}
			class="flex w-full flex-col items-center gap-4 text-center"
			onsubmit={() =>
				Submit(async () => {
					setToken((await Glue.auth.login(loginRequest)).token);
					await loadActive(); // clear any old state
					await goto(resolve('/profile'));
				}, loginState)}
		>
			<img alt="icon" src={logo} class="h-32" />
			<h1 class="text-4xl opacity-80">CloudJam Login</h1>
			<SmartField
				id="email"
				class="w-full"
				bind:value={loginRequest.email}
				placeholder="Please enter your email"
				type="email"
			/>
			<SmartField
				id="password"
				bind:value={loginRequest.password}
				placeholder="Please enter your password"
				type="password"
			/>
			<Button class="w-full" type="submit" variant="primary" processing={loginState.loading}>Login</Button>
			{#if loginState.error}
				<Alert class="w-full" title="Authentication failed" message={loginState.error} animation={true} />
			{/if}
		</SmartForm>
	</Card>
</div>

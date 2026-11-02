<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, setToken, Submit, type SubmitState } from '$lib';
	import logo from '$lib/assets/favicon.svg';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import { LoginRequestSchema } from '$lib/sdk/v1/auth/auth_pb';
	import { create } from '@bufbuild/protobuf';

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
		<form
			class="flex flex-col items-center gap-4"
			onsubmit={() =>
				Submit(async () => {
					setToken((await Glue.auth.login(loginRequest)).token);
					goto('/profile');
				}, loginState)}
		>
			<img alt="icon" src={logo} class="h-32" />
			<h1 class="text-4xl opacity-80">CloudJam Login</h1>
			<Input bind:value={loginRequest.email} placeholder="Please enter your Email" type="email" />
			<Input bind:value={loginRequest.password} placeholder="Please enter your Password" type="password" />
			<Button class="w-full" type="submit" variant="primary" processing={loginState.loading}>Login</Button>
			{#if loginState.error}
				<Alert title="Authentication failed" message={loginState.error} animation={true} />
			{/if}
		</form>
	</Card>
</div>

<script lang="ts">
	import { Glue } from '$lib';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import { UserSchema } from '$lib/sdk/v1/admin/user_pb';
	import { create } from '@bufbuild/protobuf';

	let user = $state(
		create(UserSchema, {
			username: '',
			email: ''
		})
	);

	let validator = $derived(Glue.Validate(UserSchema, user));
</script>

<h1>Create User</h1>

<form
	onsubmit={() => {
		user = user;
		console.log(user);
	}}
></form>

<Button variant="primary">Bodenlos</Button>

<Input
	bind:value={user.username}
	id="username"
	type="text"
	label="Username"
	placeholder="Enter your username"
	validation={validator.violation.username}
/>

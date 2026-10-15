export function createState<T>(initial: T) {
	let state = $state({ value: initial });
	return state;
}

export type Level = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

const colors: Record<Level, string> = {
	DEBUG: 'sky-300',
	INFO: 'emerald-300',
	WARN: 'amber-300',
	ERROR: 'red-300'
};

export const levelColors: Record<Level, string> = {
	DEBUG: `var(--color-${colors.DEBUG})`,
	INFO: `var(--color-${colors.INFO})`,
	WARN: `var(--color-${colors.WARN})`,
	ERROR: `var(--color-${colors.ERROR})`
};

export const severities: Record<Level, { short: string; level: string }> = {
	DEBUG: { short: 'DBG', level: `text-${colors.DEBUG}` },
	INFO: { short: 'INF', level: `text-${colors.INFO}` },
	WARN: { short: 'WRN', level: `text-${colors.WARN}` },
	ERROR: { short: 'ERR', level: `text-${colors.ERROR}` }
};

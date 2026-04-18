import js from "@eslint/js";
import globals from "globals";
import pluginVue from "eslint-plugin-vue";
import tseslint from "typescript-eslint";

const commonIgnores = [
	"**/node_modules/**",
	"**/dist/**",
	"**/coverage/**",
	"**/storybook-static/**",
	"**/test-results/**",
	"**/.output/**",
];

export default tseslint.config(
	{
		ignores: commonIgnores,
	},
	js.configs.recommended,
	...tseslint.configs.recommended,
	...pluginVue.configs["flat/essential"],
	{
		files: ["**/*.{ts,tsx,vue}"],
		languageOptions: {
			ecmaVersion: "latest",
			sourceType: "module",
			globals: {
				...globals.browser,
				...globals.node,
			},
			parserOptions: {
				parser: tseslint.parser,
				extraFileExtensions: [".vue"],
			},
		},
		rules: {
			"no-console": ["warn", { allow: ["info", "warn", "error"] }],
			"no-debugger": "warn",
			"no-empty": ["error", { allowEmptyCatch: false }],
			"no-undef": "off",
		},
	},
	{
		files: ["web/src/**/*.vue", "admin/src/**/*.vue"],
		rules: {
			"vue/multi-word-component-names": "off",
		},
	},
);

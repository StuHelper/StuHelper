import js from "@eslint/js";
import globals from "globals";
import pluginVue from "eslint-plugin-vue";
import pluginVueA11y from "eslint-plugin-vuejs-accessibility";
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
	...pluginVueA11y.configs["flat/recommended"],
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
			"vue/html-button-has-type": "error",
			"vuejs-accessibility/interactive-supports-focus": [
				"error",
				{
					tabbable: [
						"button",
						"checkbox",
						"combobox",
						"link",
						"menuitem",
						"radio",
						"searchbox",
						"spinbutton",
						"switch",
						"tab",
						"textbox",
					],
				},
			],
			"vuejs-accessibility/label-has-for": [
				"error",
				{
					allowChildren: true,
					controlComponents: ["Input"],
					required: { some: ["nesting", "id"] },
				},
			],
			"vuejs-accessibility/no-static-element-interactions": "off",
		},
	},
	{
		files: ["web/src/**/*.vue", "admin/src/**/*.vue"],
		rules: {
			"vue/multi-word-component-names": "off",
		},
	},
);

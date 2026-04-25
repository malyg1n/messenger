import js from "@eslint/js"
import tseslint from "@typescript-eslint/eslint-plugin"
import tsParser from "@typescript-eslint/parser"

export default [
  {
    ignores: ["dist/**", "node_modules/**"]
  },
  js.configs.recommended,
  {
    files: ["src/**/*.ts", "src/**/*.tsx"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: "latest",
        sourceType: "module"
      },
      globals: {
        document: "readonly",
        fetch: "readonly",
        HTMLDivElement: "readonly",
        localStorage: "readonly",
        requestAnimationFrame: "readonly",
        WebSocket: "readonly"
      }
    },
    plugins: {
      "@typescript-eslint": tseslint
    },
    rules: {
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": ["error"],
      "no-undef": "off"
    }
  },
  {
    files: ["src/entities/**/*.ts", "src/entities/**/*.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/app/**", "@/pages/**", "@/widgets/**", "@/features/**"],
              message: "entities layer cannot import upper layers"
            }
          ]
        }
      ]
    }
  },
  {
    files: ["src/features/**/*.ts", "src/features/**/*.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/app/**", "@/pages/**", "@/widgets/**"],
              message: "features layer cannot import upper layers"
            }
          ]
        }
      ]
    }
  },
  {
    files: ["src/widgets/**/*.ts", "src/widgets/**/*.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/app/**", "@/pages/**"],
              message: "widgets layer cannot import upper layers"
            }
          ]
        }
      ]
    }
  },
  {
    files: ["src/pages/**/*.ts", "src/pages/**/*.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/app/**"],
              message: "pages layer cannot import app layer"
            }
          ]
        }
      ]
    }
  }
]

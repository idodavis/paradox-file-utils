/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'dark-bg': '#1b2636',
        'dark-panel': '#11233f',
        'dark-input': '#131725',
        'dark-border': '#3a4551',
        'btn-primary': '#295b7e',
        'btn-primary-hover': '#3677a4',
      },
    },
  },
  plugins: [],
}


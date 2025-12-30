// A bright TypeScript Hello World with a tiny pulse of color.

const banner = (text: string) => {
  const line = "=".repeat(text.length);
  return `${line}\n${text}\n${line}`;
};

const greeting = "Hello, TypeScript world!";
console.log(banner(greeting));
console.log("Sparkles:", ["✨", "🌟", "💫"].join(" "));
console.log("Timestamp:", new Date().toISOString());

export function shellQuote(s: string): string {
  if (s === "") return "''";
  return "'" + s.replaceAll("'", "'\"'\"'") + "'";
}

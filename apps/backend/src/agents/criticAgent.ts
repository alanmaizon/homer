export class CriticAgent {
  review(result: string): string {
    if (result.trim().length < 20) {
      return `${result}\n\n[Critic] Consider adding more detail.`;
    }
    return result;
  }
}

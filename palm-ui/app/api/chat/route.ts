export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function POST(req: Request) {
  const body = await req.json();
  const { messages } = body;

  if (!messages || messages.length === 0) {
    return new Response("No messages provided", { status: 400 });
  }

  const lastMessage = messages[messages.length - 1];

  const userMessage = lastMessage?.parts
    ?.filter((part: any) => part.type === "text")
    .map((part: any) => part.text)
    .join("") || lastMessage?.content || "";

  if (!userMessage) {
    return new Response("No message provided", { status: 400 });
  }

  const palmUrl = process.env.PALM_URL || "http://localhost:4096";

  console.log("Calling Palm server:", palmUrl, "with message:", userMessage);

  try {
    const response = await fetch(`${palmUrl}/chat`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        message: userMessage,
      }),
    });

    console.log("Palm server response status:", response.status);

    if (!response.ok) {
      throw new Error(`Palm server error: ${response.statusText}`);
    }

    if (!response.body) {
      throw new Error("No response body from Palm server");
    }

    console.log("Forwarding stream to client");

    return new Response(response.body, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
        "X-Vercel-AI-Data-Stream": "v1",
      },
    });
  } catch (error) {
    console.error("Error calling Palm server:", error);
    return new Response(
      `Error: ${error instanceof Error ? error.message : "Unknown error"}`,
      { status: 500 }
    );
  }
}

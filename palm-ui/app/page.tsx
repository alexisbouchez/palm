"use client";

import * as React from "react";
import { useChat } from "@ai-sdk/react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Send, Bot, User } from "lucide-react";

export default function Home() {
  const [input, setInput] = React.useState("");
  const { messages, sendMessage, status } = useChat({
    api: "/api/chat",
    streamProtocol: "data",
  });

  const isLoading = status === "in-progress" || status === "streaming" || status === "submitted";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const messageText = input.trim();
    setInput("");
    await sendMessage({ text: messageText });
  };

  const getTextContent = (message: typeof messages[number]) => {
    return message.parts
      ?.filter((part: any) => part.type === "text")
      .map((part: any) => part.text)
      .join("") || "";
  };

  const getToolCalls = (message: typeof messages[number]) => {
    const tools = message.parts?.filter((part: any) =>
      part.type === "dynamic-tool" || part.type?.startsWith("tool-")
    ) || [];

    return tools.map((tool: any) => {
      if (tool.type?.startsWith("tool-") && !tool.toolName) {
        return {
          ...tool,
          toolName: tool.type.replace(/^tool-/, "")
        };
      }
      return tool;
    });
  };

  return (
    <main className="min-h-screen flex items-center justify-center p-4 bg-linear-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900">
      <Card className="w-full max-w-3xl h-175 flex flex-col shadow-xl">
        <CardHeader className="border-b">
          <CardTitle className="flex items-center gap-2">
            <Bot className="w-6 h-6" />
            Palm Agent
          </CardTitle>
          <CardDescription>
            AI agent powered by Mistral and Vercel AI SDK
          </CardDescription>
        </CardHeader>

        <CardContent className="flex-1 flex flex-col p-0">
          <ScrollArea className="flex-1 p-6">
            <div className="space-y-4">
              {messages.length === 0 && (
                <div className="text-center text-muted-foreground py-12">
                  <Bot className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p className="text-lg">Start a conversation with Palm</p>
                  <p className="text-sm mt-2">
                    Try asking about the weather or anything else!
                  </p>
                </div>
              )}

              {messages.map((message, index) => {
                const textContent = getTextContent(message);
                const toolCalls = getToolCalls(message);

                if (message.role === "assistant") {
                  const hasTextContent = textContent && textContent.length > 0;
                  const hasToolCalls = toolCalls && toolCalls.length > 0;

                  if (!hasTextContent && hasToolCalls) {
                    return null;
                  }

                  const onlyHasStepStart = message.parts?.every((part: any) =>
                    part.type === "step-start"
                  );

                  if (!hasTextContent && !hasToolCalls && onlyHasStepStart) {
                    return null;
                  }

                  if (!hasTextContent && !hasToolCalls && (!message.parts || message.parts.length === 0)) {
                    return null;
                  }
                }

                if (message.role === "user" && !textContent) {
                  return null;
                }

                return (
                <div
                  key={message.id}
                  className={`flex gap-3 ${
                    message.role === "assistant"
                      ? "justify-start"
                      : "justify-end"
                  }`}
                >
                  {message.role === "assistant" && (
                    <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shrink-0">
                      <Bot className="w-5 h-5 text-primary-foreground" />
                    </div>
                  )}

                  <div
                    className={`rounded-lg px-4 py-2 max-w-[80%] ${
                      message.role === "assistant"
                        ? "bg-muted"
                        : "bg-primary text-primary-foreground"
                    }`}
                  >
                    <p className="text-sm whitespace-pre-wrap">
                      {getTextContent(message)}
                    </p>

                    {toolCalls.length > 0 && (
                      <div className="mt-2 pt-2 border-t border-border/50">
                        <p className="text-xs font-medium mb-1">Tools:</p>
                        {toolCalls.map((tool: any, i: number) => (
                          <div key={i} className="text-xs mb-2 bg-muted/30 p-2 rounded">
                            <div className="flex items-center gap-2 mb-1">
                              <span className="font-mono font-semibold text-primary">
                                {tool.toolName || "unknown"}
                              </span>
                              {tool.state && (
                                <span className="text-muted-foreground text-xs opacity-70">
                                  {tool.state}
                                </span>
                              )}
                            </div>
                            {tool.input && (
                              <div className="ml-2 mt-1 text-muted-foreground">
                                Input: {JSON.stringify(tool.input)}
                              </div>
                            )}
                            {tool.output && (
                              <div className="ml-2 mt-1 text-green-600">
                                Output: {JSON.stringify(tool.output)}
                              </div>
                            )}
                            {tool.errorText && (
                              <div className="ml-2 mt-1 text-red-600">
                                Error: {tool.errorText}
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>

                  {message.role === "user" && (
                    <div className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center shrink-0">
                      <User className="w-5 h-5 text-secondary-foreground" />
                    </div>
                  )}
                </div>
              );
              })}

              {isLoading && (
                <div className="flex gap-3 justify-start">
                  <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shrink-0">
                    <Bot className="w-5 h-5 text-primary-foreground animate-pulse" />
                  </div>
                  <div className="rounded-lg px-4 py-2 bg-muted">
                    <div className="flex gap-1">
                      <div
                        className="w-2 h-2 rounded-full bg-foreground/40 animate-bounce"
                        style={{ animationDelay: "0ms" }}
                      />
                      <div
                        className="w-2 h-2 rounded-full bg-foreground/40 animate-bounce"
                        style={{ animationDelay: "150ms" }}
                      />
                      <div
                        className="w-2 h-2 rounded-full bg-foreground/40 animate-bounce"
                        style={{ animationDelay: "300ms" }}
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </ScrollArea>

          <div className="border-t p-4">
            <form onSubmit={handleSubmit} className="flex gap-2">
              <Input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Ask anything..."
                disabled={isLoading}
                className="flex-1"
              />
              <Button
                type="submit"
                disabled={isLoading || !input?.trim()}
              >
                <Send className="w-4 h-4" />
              </Button>
            </form>
          </div>
        </CardContent>
      </Card>
    </main>
  );
}

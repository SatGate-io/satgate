import Link from 'next/link';
import { ArrowLeft, ArrowRight, Calendar, Clock, User } from 'lucide-react';

export const metadata = {
  title: 'Blog - SatGate',
  description: 'Insights on API economics, agent governance, and the future of machine-to-machine commerce.',
};

// Blog post data - in a real setup this would come from a CMS or markdown files
const posts = [
  {
    slug: 'how-we-built-budget-enforcement-mcp',
    title: 'How We Built Budget Enforcement for MCP Tool Calls',
    description: '2,164 lines of Go, 28 tests, and one evening. An open-source MCP proxy that enforces per-tool budgets with cryptographic delegation.',
    date: '2026-02-13',
    readTime: '8 min read',
    author: 'SatGate Team',
    tags: ['MCP', 'Go', 'Macaroons', 'Engineering'],
  },
  {
    slug: 'beyond-connection-economic-governance-mcp',
    title: 'Beyond Connection: The Case for Economic Governance in MCP',
    description: 'The MCP ecosystem talks about capability. Nobody talks about cost. Here\'s why economic policy is the missing layer — and how to enforce it at the protocol level with L402 and macaroons.',
    date: '2026-02-12',
    readTime: '12 min read',
    author: 'SatGate Team',
    tags: ['MCP', 'L402', 'Macaroons', 'Agent Economy'],
  },
  {
    slug: 'why-routing-isnt-governance',
    title: 'Why Routing Isn\'t Governance',
    description: 'AI gateways excel at routing LLM calls. But when agents control spend autonomously, routing isn\'t enough. You need economic governance.',
    date: '2026-02-06',
    readTime: '5 min read',
    author: 'SatGate Team',
    tags: ['MCP', 'Governance', 'Agent Economy'],
  },
];

export default function BlogPage() {
  return (
    <div className="min-h-screen bg-black text-gray-100 font-sans">
      <div className="max-w-4xl mx-auto px-6 py-16">
        <Link href="/" className="text-gray-500 hover:text-white flex items-center gap-2 transition mb-8">
          <ArrowLeft size={18} /> Back to Home
        </Link>
        
        <div className="mb-12">
          <h1 className="text-4xl font-bold mb-4">Blog</h1>
          <p className="text-xl text-gray-400">
            Insights on API economics, agent governance, and the future of machine-to-machine commerce.
          </p>
        </div>

        <div className="space-y-6">
          {posts.map((post) => (
            <Link 
              key={post.slug}
              href={`/blog/${post.slug}`}
              className="block bg-gray-900 border border-gray-800 rounded-xl p-6 hover:border-purple-600/50 transition group"
            >
              <div className="flex flex-wrap gap-2 mb-3">
                {post.tags.map((tag) => (
                  <span 
                    key={tag}
                    className="px-2 py-1 rounded-full bg-purple-900/30 border border-purple-500/30 text-purple-300 text-xs font-mono"
                  >
                    {tag}
                  </span>
                ))}
              </div>
              
              <h2 className="text-xl font-bold mb-2 group-hover:text-purple-400 transition">
                {post.title}
              </h2>
              
              <p className="text-gray-400 mb-4">
                {post.description}
              </p>
              
              <div className="flex items-center gap-4 text-sm text-gray-500">
                <span className="flex items-center gap-1">
                  <Calendar size={14} />
                  {new Date(post.date).toLocaleDateString('en-US', { 
                    month: 'short', 
                    day: 'numeric', 
                    year: 'numeric' 
                  })}
                </span>
                <span className="flex items-center gap-1">
                  <Clock size={14} />
                  {post.readTime}
                </span>
              </div>
              
              <div className="mt-4 flex items-center text-purple-400 text-sm font-medium group-hover:gap-2 transition-all">
                Read more <ArrowRight size={16} className="ml-1" />
              </div>
            </Link>
          ))}
        </div>

        {/* Newsletter CTA */}
        <section className="mt-16 bg-gradient-to-r from-purple-900/20 to-cyan-900/20 border border-purple-800/30 rounded-xl p-8 text-center">
          <h2 className="text-2xl font-bold mb-3">Stay Updated</h2>
          <p className="text-gray-400 mb-6">
            Get insights on API economics and agent governance delivered to your inbox.
          </p>
          <a 
            href="mailto:contact@satgate.io?subject=Newsletter%20Subscription"
            className="inline-flex items-center gap-2 bg-white text-black px-6 py-3 rounded-lg font-bold hover:bg-gray-200 transition"
          >
            Subscribe to Updates
          </a>
        </section>
      </div>
    </div>
  );
}

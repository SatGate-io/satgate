'use client';

import { useState } from 'react';
import { Menu, X } from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';

export default function SiteMenu({ signupUrl }: { signupUrl: string }) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <nav className="border-b border-gray-800 backdrop-blur-md relative w-full z-50 bg-black/50">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2 shrink-0 min-h-11">
          <Image src="/logo_white_transparent.png" alt="SatGate" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
          <span className="text-lg sm:text-xl font-bold text-white whitespace-nowrap">SatGate<sup className="text-xs font-normal">TM</sup></span>
        </Link>

        {/* Desktop menu */}
        <div className="hidden xl:flex items-center gap-5 text-sm font-medium text-gray-400">
          <Link href="/platform" className="hover:text-white transition">Platform</Link>
          <Link href="/govern" className="hover:text-white transition">Enterprise</Link>
          <Link href="/mcp" className="hover:text-white transition">MCP</Link>
          <Link href="/build" className="hover:text-white transition">Build</Link>
          <Link href="/sandbox" className="hover:text-white transition">Demo</Link>
          <Link href="/pricing" className="hover:text-white transition">Pricing</Link>
          <a href="https://cloud.satgate.io/docs" target="_blank" rel="noopener noreferrer" className="hover:text-white transition">Docs</a>
          <a href={signupUrl} target="_blank" rel="noopener noreferrer" className="rounded-full bg-white px-3 py-1.5 font-bold text-black hover:bg-gray-200 transition">Start free trial</a>
          <a href="https://cloud.satgate.io/cloud/login" target="_blank" rel="noopener noreferrer" className="rounded-full border border-purple-500/40 px-3 py-1.5 text-purple-300 hover:border-purple-400 hover:text-purple-200 transition">Cloud login</a>
        </div>

        {/* Mobile menu button */}
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="flex xl:hidden items-center justify-center w-11 h-11 rounded-lg bg-gray-800/50 hover:bg-gray-700/50 text-gray-400 hover:text-white transition"
          aria-label="Toggle menu"
          aria-expanded={mobileMenuOpen}
        >
          {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
        </button>
      </div>

      {/* Mobile menu dropdown */}
      <div
        className={`xl:hidden transition-all duration-300 ease-in-out ${
          mobileMenuOpen ? 'max-h-[calc(100vh-4rem)] overflow-y-auto opacity-100' : 'max-h-0 overflow-hidden opacity-0'
        }`}
      >
        <div className="bg-black/95 backdrop-blur-xl border-t border-gray-800 px-4 py-4 space-y-1">
          <Link
            href="/platform"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Platform
          </Link>
          <Link
            href="/govern"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Enterprise
          </Link>
          <Link
            href="/mcp"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            MCP
          </Link>
          <Link
            href="/build"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Build
          </Link>
          <Link
            href="/sandbox"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Demo
          </Link>
          <Link
            href="/security"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Security
          </Link>
          <Link
            href="/pricing"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Pricing
          </Link>
          <Link
            href="/tools"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Tools
          </Link>
          <Link
            href="/blog"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Blog
          </Link>
          <a
            href="https://cloud.satgate.io/docs"
            target="_blank"
            rel="noopener noreferrer"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Docs
          </a>
          <a
            href="https://github.com/SatGate-io/satgate"
            target="_blank"
            rel="noopener noreferrer"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            GitHub
          </a>
          <a
            href={signupUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={() => setMobileMenuOpen(false)}
            className="block bg-white text-black font-bold transition py-3 px-4 rounded-lg"
          >
            Start free trial
          </a>
          <a
            href="https://cloud.satgate.io/cloud/login"
            target="_blank"
            rel="noopener noreferrer"
            onClick={() => setMobileMenuOpen(false)}
            className="block text-gray-400 hover:text-white hover:bg-gray-800/50 transition py-3 px-4 rounded-lg"
          >
            Cloud login
          </a>
        </div>
      </div>
    </nav>
  );
}

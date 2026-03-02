import { NextRequest, NextResponse } from 'next/server';

async function sendEmail(apiKey: string, params: { from: string; to: string[]; cc?: string[]; subject: string; html: string }) {
  const res = await fetch('https://api.resend.com/emails', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(params),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`Resend API error ${res.status}: ${err}`);
  }
  return res.json();
}

export async function POST(req: NextRequest) {
  try {
    const data = await req.json();
    const { name, email, company, role, agentCount, apis, challenge } = data;

    if (!name || !email || !company) {
      return NextResponse.json({ error: 'Missing required fields' }, { status: 400 });
    }

    const apiKey = process.env.RESEND_API_KEY;
    if (!apiKey) {
      console.error('[Design Partner] RESEND_API_KEY not configured');
      // Still log the submission even if email fails
      console.log(`[Design Partner] ${company} — ${name} (${email})`);
      return NextResponse.json({ ok: true, message: "Application received! We'll be in touch within 24 hours." });
    }

    const fromName = process.env.RESEND_FROM_NAME || 'SatGate';
    const fromEmail = process.env.RESEND_FROM_EMAIL || 'noreply@satgate.io';

    // Send notification to Wayne + Matt
    await sendEmail(apiKey, {
      from: `${fromName} <${fromEmail}>`,
      to: ['wayne@satgate.io'],
      cc: ['matt@satgate.io'],
      subject: `🚀 New Design Partner Application: ${company}`,
      html: `
        <div style="font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 0 auto;">
          <h2 style="color: #7c3aed;">New Design Partner Application</h2>
          <table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
            <tr><td style="padding: 8px 12px; font-weight: bold; color: #666; width: 140px;">Name</td><td style="padding: 8px 12px;">${name}</td></tr>
            <tr style="background: #f9fafb;"><td style="padding: 8px 12px; font-weight: bold; color: #666;">Email</td><td style="padding: 8px 12px;"><a href="mailto:${email}">${email}</a></td></tr>
            <tr><td style="padding: 8px 12px; font-weight: bold; color: #666;">Company</td><td style="padding: 8px 12px;">${company}</td></tr>
            <tr style="background: #f9fafb;"><td style="padding: 8px 12px; font-weight: bold; color: #666;">Role</td><td style="padding: 8px 12px;">${role || 'Not specified'}</td></tr>
            <tr><td style="padding: 8px 12px; font-weight: bold; color: #666;">AI Agents</td><td style="padding: 8px 12px;">${agentCount || 'Not specified'}</td></tr>
            <tr style="background: #f9fafb;"><td style="padding: 8px 12px; font-weight: bold; color: #666;">APIs</td><td style="padding: 8px 12px;">${(apis || []).join(', ') || 'None selected'}</td></tr>
          </table>
          ${challenge ? `
            <div style="margin: 20px 0; padding: 16px; background: #f3f4f6; border-radius: 8px;">
              <p style="margin: 0 0 8px 0; font-weight: bold; color: #666;">Biggest Challenge:</p>
              <p style="margin: 0; color: #111;">${challenge}</p>
            </div>
          ` : ''}
          <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 24px 0;" />
          <p style="color: #9ca3af; font-size: 12px;">Submitted via satgate.io/design-partners at ${new Date().toISOString()}</p>
        </div>
      `,
    });

    // Send confirmation to applicant
    await sendEmail(apiKey, {
      from: `${fromName} <${fromEmail}>`,
      to: [email],
      subject: 'SatGate Design Partner Application Received',
      html: `
        <div style="font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 0 auto;">
          <h2>Thanks for applying, ${(name || '').split(' ')[0]}!</h2>
          <p>We received your Design Partner application for <strong>${company}</strong>.</p>
          <p>We review applications within 24 hours. You'll hear from Wayne Mattadeen, our founder, directly.</p>
          <p>In the meantime, you can:</p>
          <ul>
            <li><a href="https://cloud.satgate.io/cloud/login">Sign up for SatGate Cloud</a> (free Observe mode)</li>
            <li><a href="https://cloud.satgate.io/docs">Browse the docs</a></li>
            <li><a href="https://github.com/SatGate-io/satgate">Check out the open source gateway</a></li>
          </ul>
          <p>Looking forward to working together.</p>
          <p style="color: #666;">— The SatGate Team</p>
        </div>
      `,
    });

    console.log(`[Design Partner] ✅ ${company} — ${name} (${email})`);
    return NextResponse.json({ ok: true, message: "Application received! We'll be in touch within 24 hours." });
  } catch (err) {
    console.error('[Design Partner] Error:', err);
    return NextResponse.json({ error: 'Internal error' }, { status: 500 });
  }
}

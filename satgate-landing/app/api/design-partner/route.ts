import { NextResponse } from 'next/server';
import { Resend } from 'resend';

// Lazy init to avoid build-time crash when env var is missing
let _resend: Resend | null = null;
function getResend() {
  if (!_resend) {
    if (!process.env.RESEND_API_KEY) throw new Error('RESEND_API_KEY not configured');
    _resend = new Resend(process.env.RESEND_API_KEY);
  }
  return _resend;
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const { name, email, company, role, agentCount, apis, challenge } = body;

    if (!name || !email || !company || !role) {
      return NextResponse.json({ error: 'Missing required fields' }, { status: 400 });
    }

    // Send notification to Wayne
    await getResend().emails.send({
      from: `${process.env.RESEND_FROM_NAME || 'SatGate'} <${process.env.RESEND_FROM_EMAIL || 'noreply@satgate.io'}>`,
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
            <tr style="background: #f9fafb;"><td style="padding: 8px 12px; font-weight: bold; color: #666;">Role</td><td style="padding: 8px 12px;">${role}</td></tr>
            <tr><td style="padding: 8px 12px; font-weight: bold; color: #666;">AI Agents</td><td style="padding: 8px 12px;">${agentCount || 'Not specified'}</td></tr>
            <tr style="background: #f9fafb;"><td style="padding: 8px 12px; font-weight: bold; color: #666;">APIs</td><td style="padding: 8px 12px;">${apis?.join(', ') || 'None selected'}</td></tr>
          </table>
          ${challenge ? `
            <div style="margin: 20px 0; padding: 16px; background: #f3f4f6; border-radius: 8px;">
              <p style="margin: 0 0 8px 0; font-weight: bold; color: #666;">Biggest Challenge:</p>
              <p style="margin: 0; color: #111;">${challenge}</p>
            </div>
          ` : ''}
          <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 24px 0;" />
          <p style="color: #9ca3af; font-size: 12px;">This application was submitted via satgate.io/design-partners</p>
        </div>
      `,
    });

    // Send confirmation to applicant
    await getResend().emails.send({
      from: `${process.env.RESEND_FROM_NAME || 'SatGate'} <${process.env.RESEND_FROM_EMAIL || 'noreply@satgate.io'}>`,
      to: [email],
      subject: 'SatGate Design Partner Application Received',
      html: `
        <div style="font-family: system-ui, -apple-system, sans-serif; max-width: 600px; margin: 0 auto;">
          <h2>Thanks for applying, ${name.split(' ')[0]}!</h2>
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

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Design partner submission error:', error);
    return NextResponse.json({ error: 'Failed to submit application' }, { status: 500 });
  }
}

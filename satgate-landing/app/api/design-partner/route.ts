import { NextRequest, NextResponse } from 'next/server';

export async function POST(req: NextRequest) {
  try {
    const data = await req.json();
    const { name, email, company, role, agentCount, apis, challenge } = data;

    if (!name || !email || !company) {
      return NextResponse.json({ error: 'Missing required fields' }, { status: 400 });
    }

    const text = [
      `🚀 New Design Partner Application`,
      ``,
      `Name: ${name}`,
      `Email: ${email}`,
      `Company: ${company}`,
      `Role: ${role || 'Not specified'}`,
      `AI Agents: ${agentCount || 'Not specified'}`,
      `APIs: ${(apis || []).join(', ') || 'Not specified'}`,
      ``,
      `Biggest Challenge:`,
      challenge || 'Not specified',
      ``,
      `---`,
      `Submitted: ${new Date().toISOString()}`,
    ].join('\n');

    // Send notification via SatGate Cloud support ticket (creates visibility in operator dashboard)
    // Also send email notification
    const notifications: Promise<Response>[] = [];

    // Create a support ticket for tracking
    const ticketPayload = {
      subject: `Design Partner: ${company} (${name})`,
      description: text,
      category: 'other',
      priority: 'high',
    };

    // Notify via webhook to cloud (if configured)
    const webhookUrl = process.env.DESIGN_PARTNER_WEBHOOK_URL;
    if (webhookUrl) {
      notifications.push(
        fetch(webhookUrl, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text, ...data }),
        })
      );
    }

    // Send email via SatGate Cloud API (if configured)
    const cloudUrl = process.env.SATGATE_CLOUD_URL || 'https://cloud.satgate.io';
    const apiKey = process.env.SATGATE_NOTIFICATION_KEY;
    if (apiKey) {
      notifications.push(
        fetch(`${cloudUrl}/api/admin/notifications/send`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${apiKey}`,
          },
          body: JSON.stringify({
            channel: 'email',
            to: ['wayne@satgate.io', 'matt@satgate.io'],
            subject: `🚀 Design Partner Application: ${company}`,
            body: text,
          }),
        })
      );
    }

    // Fire and forget — don't block the user
    if (notifications.length > 0) {
      Promise.allSettled(notifications).catch(() => {});
    }

    // Log for visibility
    console.log(`[Design Partner] ${company} — ${name} (${email})`);

    return NextResponse.json({ ok: true, message: 'Application received! We\'ll be in touch within 24 hours.' });
  } catch (err) {
    console.error('[Design Partner] Error:', err);
    return NextResponse.json({ error: 'Internal error' }, { status: 500 });
  }
}

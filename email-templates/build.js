// Charger Babel pour compiler JSX
require('@babel/register')({
  presets: ['@babel/preset-react'],
  extensions: ['.jsx', '.js']
});

const { render } = require('@react-email/render');
const fs = require('fs');
const path = require('path');
const React = require('react');

// Importer les templates
const WelcomeEmail = require('./emails/welcome.jsx').default;
const OrderConfirmationEmail = require('./emails/order-confirmation.jsx').default;

// Créer le dossier de sortie s'il n'existe pas
const outputDir = path.join(__dirname, '../internal/templates');
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

async function buildTemplates() {
  console.log('🔨 Compilation des templates email...\n');

  // 1. Welcome Email
  console.log('📧 Compilation: welcome.html');
  const welcomeHtml = await render(
    React.createElement(WelcomeEmail, { userName: '{{.UserName}}' })
  );
  fs.writeFileSync(
    path.join(outputDir, 'welcome.html'),
    welcomeHtml
  );
  console.log('✅ welcome.html créé\n');

  // 2. Order Confirmation Email
  console.log('📧 Compilation: order-confirmation.html');
  const orderHtml = await render(
    React.createElement(OrderConfirmationEmail, { 
      orderID: '{{.OrderID}}',
      totalAmount: '{{.TotalAmount}}'
    })
  );
  fs.writeFileSync(
    path.join(outputDir, 'order-confirmation.html'),
    orderHtml
  );
  console.log('✅ order-confirmation.html créé\n');

  console.log('🎉 Tous les templates ont été compilés avec succès !');
  console.log(`📁 Fichiers générés dans: ${outputDir}`);
}

buildTemplates().catch(console.error);

import React, {useEffect} from 'react';
import Layout from '@theme/Layout';

export default function ApiReferencePage(): React.JSX.Element {
  useEffect(() => {
    window.location.replace('/swagger/index.html');
  }, []);

  return (
    <Layout title="API Reference" description="Redirecting to the live Swagger UI reference.">
      <main style={{padding: '2rem'}}>
        <p>Redirecting to the live Swagger UI reference at <code>/swagger/index.html</code>.</p>
      </main>
    </Layout>
  );
}

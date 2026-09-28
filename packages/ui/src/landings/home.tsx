import React from 'react';
import products from '../../config/src/products';
import React from 'react';import products from '../../config/src/products';const clientProducts = products.map(product => {
    return <li key={product.id}>{product.bullets.join(', ')}</li>;
});

<ul>{clientProducts}</ul>
const Home = () => {
    return (
        <div style={{ backgroundColor: '#0c1014', color: '#d5dde3', display: 'flex', height: '100vh' }}>
            <div style={{ flex: 1, borderRight: '1px solid #7a848c', padding: '10px' }}>
                <h2>Repositories/Sessions</h2>
                <div style={{ margin: '5px', padding: '10px', border: '1px solid #7a848c', borderRadius: '5px' }}>Repo 1</div>
                <div style={{ margin: '5px', padding: '10px', border: '1px solid #7a848c', borderRadius: '5px' }}>Repo 2</div>
                <div style={{ margin: '5px', padding: '10px', border: '1px solid #7a848c', borderRadius: '5px' }}>Session 1</div>
            </div>
            <div style={{ flex: 2, padding: '10px', display: 'flex', flexDirection: 'column' }}>
                <div style={{ flex: 1 }}>
                    <h2>Chat Transcript</h2>
                    <div style={{ border: '1px solid #7a848c', borderRadius: '5px', padding: '10px', height: '80vh', overflowY: 'auto' }}>
                        <p>User: Hello there!</p>
                        <p>Agent: Hello! How can I assist you today?</p>
                    </div>
                </div>
                <div style={{ flex: 1 }}>
                    <h2>Overview</h2>
                    <div style={{ border: '1px solid #7a848c', borderRadius: '5px', padding: '10px', height: '100%', overflowY: 'auto' }}>
                        <h3>Layout Overview</h3>
                        <p>This section provides a visual layout mock of the workspace.</p>
                        <div style={{ display: 'flex', flexDirection: 'column' }}>
                            <div style={{ height: '30%', backgroundColor: '#2e353a', borderRadius: '5px', margin: '10px 0' }}>
                                <p>Visual Representation of the Web Workspace Layout</p>
                                <p>Left: Repositories/Sessions</p>
                                <p>Center: Chat Transcript</p>
                                <p>Right: Agents/Changes/Files</p>
                                <p>Design accommodates TUI and Web surfaces.</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div style={{ flex: 1, borderLeft: '1px solid #7a848c', padding: '10px' }}>
                <h2>Agents/Changes/Files</h2>
                <div style={{ margin: '5px', padding: '10px', border: '1px solid #7a848c', borderRadius: '5px' }}>Agent 1</div>
                <div style={{ margin: '5px', padding: '10px', border: '1px solid #7a848c', borderRadius: '5px' }}>Changes</div>
                <div style={{ margin: '5px', padding: '10px', border: '1px solid #7a848c', borderRadius: '5px' }}>File 1</div>
            </div>
        </div>
    );
};

export default Home;
